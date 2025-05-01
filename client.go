package client

import (
	"context"
	"github.com/robotjoosen/go-tplink-easysmart-client/internal/udp"

	"github.com/robotjoosen/go-tplink-easysmart-client/internal/gui"
	"github.com/robotjoosen/go-tplink-easysmart-client/pkg/model"
)

type mode string

const (
	GUI mode = "gui"
	UDP mode = "udp"
)

type ClientAware interface {
	Login(ctx context.Context, username, password string) error
	GetDevice(ctx context.Context) (model.Device, error)
	GetPortStats(ctx context.Context) ([]model.Port, error)
	SetLED(ctx context.Context, state bool) error
}

func NewClient(address string, m mode) ClientAware {
	switch m {
	case GUI:
		return gui.NewClient(address)
	case UDP:
		return udp.NewClient(address)
	default:
		return nil
	}
}
