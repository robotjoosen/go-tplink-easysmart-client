package tlpp

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	clientErrors "github.com/robotjoosen/go-tplink-easysmart-client/pkg/errors"
	"github.com/robotjoosen/go-tplink-easysmart-client/pkg/model"
)

const (
	networkUDP = "udp"

	broadcastAddr = "255.255.255.255"
	tlppTXPort    = 29808
	tlppRXPort    = 29809
)

const (
	MagicLen   = 4
	HeaderLen  = 12
	portRecLen = 18
	MaxRetries = 3

	readTimeout = 2 * time.Second
)

type Command uint16

const (
	CmdDiscovery   Command = 0x0100
	CmdLogin       Command = 0x0101
	CmdLogout      Command = 0x0102
	CmdGetDevice   Command = 0x0200
	CmdGetPortStat Command = 0x0300
	CmdSetPortStat Command = 0x0301
	CmdSetLED      Command = 0x0400
)

type Header struct {
	Magic    [MagicLen]byte
	Length   uint16
	Sequence uint32
	Command  uint16
}

type Client struct {
	mu       sync.Mutex // guards conn, sequence and session across goroutines
	conn     *net.UDPConn
	addr     *net.UDPAddr
	sequence uint32
	session  []byte
}

func NewClient(address string) *Client {
	return &Client{
		addr: &net.UDPAddr{
			IP:   net.ParseIP(address),
			Port: tlppTXPort,
		},
	}
}

func (c *Client) Connect() error {
	conn, err := net.DialUDP(networkUDP, nil, c.addr)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

// roundTrip builds and sends one request and returns the reply that carries
// its sequence number. Calls are serialised because the socket is shared.
func (c *Client) roundTrip(ctx context.Context, cmd Command, payload []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := c.buildRequest(cmd, payload)

	return c.send(ctx, req, binary.BigEndian.Uint32(req[6:10]))
}

func (c *Client) send(ctx context.Context, req []byte, seq uint32) ([]byte, error) {
	if c.conn == nil {
		if err := c.Connect(); err != nil {
			return nil, err
		}
	}

	buf := make([]byte, 4096)
	for i := 0; i < MaxRetries; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if _, err := c.conn.Write(req); err != nil {
			return nil, err
		}

		deadline := time.Now().Add(readTimeout)
		if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
			deadline = d
		}
		_ = c.conn.SetReadDeadline(deadline)

		for {
			n, _, err := c.conn.ReadFromUDP(buf)
			if err != nil {
				break // timed out: resend
			}

			// replies to earlier retries carry an older sequence; skip them
			if n >= HeaderLen && binary.BigEndian.Uint32(buf[6:10]) == seq {
				return append([]byte(nil), buf[:n]...), nil
			}
		}
	}

	return nil, errors.New("no response from switch")
}

func (c *Client) Login(ctx context.Context, username, password string) error {
	resp, err := c.roundTrip(ctx, CmdLogin, c.buildLoginPayload(username, password))
	if err != nil {
		return errors.Join(err, clientErrors.ErrAuthenticationFailed)
	}

	if err := c.parseResponse(resp, CmdLogin); err != nil {
		return err
	}

	if len(resp) < HeaderLen+17 {
		return clientErrors.ErrAuthenticationFailed
	}

	status := resp[HeaderLen]
	if status != 0 {
		return clientErrors.ErrAuthenticationFailed
	}

	c.mu.Lock()
	c.session = append([]byte(nil), resp[HeaderLen+1:HeaderLen+17]...)
	c.mu.Unlock()

	return nil
}

func (c *Client) GetDevice(ctx context.Context) (model.Device, error) {
	resp, err := c.roundTrip(ctx, CmdGetDevice, nil)
	if err != nil {
		return model.Device{}, errors.Join(err, clientErrors.ErrInvalidResponse)
	}

	if err := c.parseResponse(resp, CmdGetDevice); err != nil {
		return model.Device{}, err
	}

	return c.parseDeviceInfo(resp[HeaderLen:])
}

func (c *Client) GetPortStats(ctx context.Context) ([]model.Port, error) {
	resp, err := c.roundTrip(ctx, CmdGetPortStat, nil)
	if err != nil {
		return nil, errors.Join(err, clientErrors.ErrInvalidResponse)
	}

	if err := c.parseResponse(resp, CmdGetPortStat); err != nil {
		return nil, err
	}

	return c.parsePortStats(resp[HeaderLen:])
}

func (c *Client) SetLED(ctx context.Context, state bool) error {
	payload := []byte{0x00}
	if state {
		payload[0] = 0x01
	}

	resp, err := c.roundTrip(ctx, CmdSetLED, payload)
	if err != nil {
		return errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	return c.parseResponse(resp, CmdSetLED)
}

func (c *Client) Logout(ctx context.Context) error {
	c.mu.Lock()
	loggedIn := c.session != nil
	c.mu.Unlock()

	if !loggedIn {
		return nil
	}

	if _, err := c.roundTrip(ctx, CmdLogout, nil); err != nil {
		return errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	c.mu.Lock()
	c.session = nil
	c.mu.Unlock()

	return nil
}

func (c *Client) SetPortState(ctx context.Context, port int, enabled bool) error {
	if err := c.validatePort(port); err != nil {
		return err
	}

	payload := make([]byte, 7)
	payload[0] = byte(port)
	if enabled {
		payload[1] = 0x01
	} else {
		payload[1] = 0x00
	}

	resp, err := c.roundTrip(ctx, CmdSetPortStat, payload)
	if err != nil {
		return errors.Join(err, clientErrors.ErrInvalidRequest)
	}

	return c.parseResponse(resp, CmdSetPortStat)
}

func (c *Client) validatePort(port int) error {
	if port < 1 || port > 24 {
		return errors.Join(errors.New("port must be between 1 and 24"), clientErrors.ErrInvalidRequest)
	}
	return nil
}

func (c *Client) ClearPortStatistics(ctx context.Context) error {
	return clientErrors.ErrNotSupported
}

func (c *Client) GetPortSpeeds(ctx context.Context) ([]model.PortSpeed, error) {
	return nil, clientErrors.ErrNotSupported
}

func (c *Client) SetPortSpeeds(ctx context.Context, port, ingress, egress int) error {
	return clientErrors.ErrNotSupported
}

func (c *Client) GetCableDiagnostics(ctx context.Context) ([]model.CableDiagnostics, error) {
	return nil, clientErrors.ErrNotSupported
}

func (c *Client) buildRequest(cmd Command, payload []byte) []byte {
	c.sequence++

	length := HeaderLen + len(payload)
	if c.session != nil {
		length += 16
	}

	header := Header{
		Magic:    [MagicLen]byte{0x00, 0x00, 0x00, 0x00},
		Length:   uint16(length),
		Sequence: c.sequence,
		Command:  uint16(cmd),
	}

	buf := make([]byte, length)
	binary.BigEndian.PutUint32(buf[0:4], binary.BigEndian.Uint32(header.Magic[:]))
	binary.BigEndian.PutUint16(buf[4:6], header.Length)
	binary.BigEndian.PutUint32(buf[6:10], header.Sequence)
	binary.BigEndian.PutUint16(buf[10:12], header.Command)

	offset := HeaderLen
	if c.session != nil {
		copy(buf[offset:offset+16], c.session)
		offset += 16
	}

	if payload != nil {
		copy(buf[offset:], payload)
	}

	return buf
}

func (c *Client) parseResponse(resp []byte, expectedCmd Command) error {
	if len(resp) < HeaderLen {
		return clientErrors.ErrInvalidResponse
	}

	length := binary.BigEndian.Uint16(resp[4:6])
	if int(length) > len(resp) {
		return clientErrors.ErrInvalidResponse
	}

	cmd := Command(binary.BigEndian.Uint16(resp[10:12]))
	if cmd != expectedCmd {
		return clientErrors.ErrInvalidResponse
	}

	return nil
}

func (c *Client) buildLoginPayload(username, password string) []byte {
	payload := make([]byte, 32)

	copy(payload[0:16], []byte(username))
	copy(payload[16:32], []byte(password))

	return payload
}

func (c *Client) parseDeviceInfo(data []byte) (model.Device, error) {
	if len(data) < 118 {
		return model.Device{}, clientErrors.ErrInvalidResponse
	}

	device := model.Device{}

	device.Description = c.readString(data[0:32])
	device.MAC = c.formatMAC(data[32:38])
	device.IP = c.formatIP(data[38:42])
	device.SubnetMask = c.formatIP(data[42:46])
	device.DefaultGateway = c.formatIP(data[46:50])
	device.FirmwareVersion = c.readString(data[50:82])
	device.HardwareVersion = c.readString(data[82:114])

	return device, nil
}

func (c *Client) parsePortStats(data []byte) ([]model.Port, error) {
	if len(data) < 1 {
		return nil, clientErrors.ErrInvalidResponse
	}

	portCount := int(data[0])
	ports := make([]model.Port, portCount)

	offset := 1
	for i := 0; i < portCount; i++ {
		if offset+portRecLen > len(data) {
			ports = ports[:i]
			break
		}

		portNum := int(data[offset])
		linkStatus := data[offset+1]

		ports[i] = model.Port{
			Number: portNum,
			Label:  fmt.Sprintf("port %d", portNum),
			Stats: model.Stats{
				Enabled:          linkStatus != 0,
				LinkStatus:       c.linkStatusToString(linkStatus),
				LinkStatusNumber: int(linkStatus),
				Packets: model.Packets{
					Transmitter: model.Transfer{
						Good: int(binary.BigEndian.Uint32(data[offset+2 : offset+6])),
						Bad:  int(binary.BigEndian.Uint32(data[offset+6 : offset+10])),
					},
					Receiver: model.Transfer{
						Good: int(binary.BigEndian.Uint32(data[offset+10 : offset+14])),
						Bad:  int(binary.BigEndian.Uint32(data[offset+14 : offset+18])),
					},
				},
			},
		}

		offset += portRecLen
	}

	return ports, nil
}

func (c *Client) readString(data []byte) string {
	for i, b := range data {
		if b == 0 {
			return string(data[:i])
		}
	}
	return string(data)
}

func (c *Client) formatMAC(data []byte) string {
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X",
		data[0], data[1], data[2], data[3], data[4], data[5])
}

func (c *Client) formatIP(data []byte) string {
	return fmt.Sprintf("%d.%d.%d.%d", data[0], data[1], data[2], data[3])
}

func (c *Client) linkStatusToString(status byte) string {
	switch status {
	case 0:
		return "Link Down"
	case 1:
		return "Auto"
	case 2:
		return "10M Half"
	case 3:
		return "10M Full"
	case 4:
		return "100M Half"
	case 5:
		return "100M Full"
	case 6:
		return "1000M Full"
	default:
		return "Unknown"
	}
}
