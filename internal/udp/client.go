package udp

import (
	"context"
	"fmt"
	"github.com/robotjoosen/go-tplink-easysmart-client/pkg/model"
	"net"
)

const (
	networkUDP = "udp"

	broadcastAddr = "255.255.255.255"
	udpTXPort     = 29808
	udpRXPort     = 29809
)

type Client struct {
	connection net.Conn
	address    string
}

func NewClient(address string) *Client {
	dial, err := net.Dial(networkUDP, fmt.Sprintf("%s:%d", broadcastAddr, udpTXPort))
	if err != nil {
		return nil
	}

	return &Client{
		connection: dial,
		address:    address,
	}
}

func (c *Client) Login(ctx context.Context, username, password string) error {
	c.connection.Write([]byte{})

	return nil
}

func (c *Client) GetDevice(ctx context.Context) (model.Device, error) { return model.Device{}, nil }

func (c *Client) GetPortStats(ctx context.Context) ([]model.Port, error) { return nil, nil }

func (c *Client) SetLED(ctx context.Context, state bool) error { return nil }
