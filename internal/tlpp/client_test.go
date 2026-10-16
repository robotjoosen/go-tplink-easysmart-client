package tlpp

import (
	"context"
	"encoding/binary"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	clientErrors "github.com/robotjoosen/go-tplink-easysmart-client/pkg/errors"
)

func Test_GetCableDiagnostics_NotSupported(t *testing.T) {
	c := NewClient("127.0.0.1")

	_, err := c.GetCableDiagnostics(context.Background())

	assert.ErrorIs(t, err, clientErrors.ErrNotSupported)
}

func Test_buildRequest_WritesCommand(t *testing.T) {
	c := NewClient("127.0.0.1")

	req := c.buildRequest(CmdSetLED, []byte{1})

	assert.Equal(t, uint16(CmdSetLED), binary.BigEndian.Uint16(req[10:12]))
	assert.Equal(t, uint16(HeaderLen+1), binary.BigEndian.Uint16(req[4:6]))
	assert.Equal(t, byte(1), req[HeaderLen])
}

func Test_parsePortStats_TwoPorts(t *testing.T) {
	c := NewClient("127.0.0.1")

	rec := func(port, link byte, tx byte) []byte {
		r := make([]byte, portRecLen)
		r[0], r[1], r[5] = port, link, tx
		return r
	}
	data := append([]byte{2}, rec(1, 5, 7)...)
	data = append(data, rec(2, 0, 9)...)

	ports, err := c.parsePortStats(data)

	assert.NoError(t, err)
	assert.Len(t, ports, 2)
	assert.Equal(t, 2, ports[1].Number)
	assert.Equal(t, "100M Full", ports[0].Stats.LinkStatus)
	assert.Equal(t, 7, ports[0].Stats.Packets.Transmitter.Good)
	assert.Equal(t, 9, ports[1].Stats.Packets.Transmitter.Good)
}

func reply(seq uint32, cmd Command) []byte {
	b := make([]byte, HeaderLen)
	binary.BigEndian.PutUint16(b[4:6], HeaderLen)
	binary.BigEndian.PutUint32(b[6:10], seq)
	binary.BigEndian.PutUint16(b[10:12], uint16(cmd))
	return b
}

func fakeSwitch(t *testing.T, handle func(conn *net.UDPConn, from *net.UDPAddr, req []byte)) *Client {
	t.Helper()

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	assert.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	go func() {
		buf := make([]byte, 4096)
		for {
			n, from, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			handle(conn, from, buf[:n])
		}
	}()

	c := NewClient("127.0.0.1")
	c.addr = conn.LocalAddr().(*net.UDPAddr)

	return c
}

func Test_send_SkipsStaleReplies(t *testing.T) {
	c := fakeSwitch(t, func(conn *net.UDPConn, from *net.UDPAddr, req []byte) {
		seq := binary.BigEndian.Uint32(req[6:10])
		_, _ = conn.WriteToUDP(reply(seq-1, CmdSetLED), from)
		_, _ = conn.WriteToUDP(reply(seq, CmdSetLED), from)
	})

	assert.NoError(t, c.SetLED(context.Background(), true))
}

func Test_send_HonoursContextDeadline(t *testing.T) {
	c := fakeSwitch(t, func(*net.UDPConn, *net.UDPAddr, []byte) {})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := c.SetLED(ctx, true)

	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), time.Second)
}

func Test_roundTrip_Concurrent(t *testing.T) {
	c := fakeSwitch(t, func(conn *net.UDPConn, from *net.UDPAddr, req []byte) {
		_, _ = conn.WriteToUDP(reply(binary.BigEndian.Uint32(req[6:10]), CmdSetLED), from)
	})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NoError(t, c.SetLED(context.Background(), true))
		}()
	}
	wg.Wait()
}
