package tlpp

import (
	"context"
	"encoding/binary"
	"testing"

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
