package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/robotjoosen/go-tplink-easysmart-client/internal/gui"
	"github.com/robotjoosen/go-tplink-easysmart-client/internal/tlpp"
	clientErrors "github.com/robotjoosen/go-tplink-easysmart-client/pkg/errors"
	"github.com/robotjoosen/go-tplink-easysmart-client/pkg/model"
)

type mode string

const (
	GUI  mode = "gui"
	TLPP mode = "tlpp"
)

type ClientAware interface {
	Login(ctx context.Context, username, password string) error
	Logout(ctx context.Context) error
	GetDevice(ctx context.Context) (model.Device, error)
	GetPortStats(ctx context.Context) ([]model.Port, error)
	SetLED(ctx context.Context, state bool) error
	SetPortState(ctx context.Context, port int, enabled bool) error
	ClearPortStatistics(ctx context.Context) error
	GetPortSpeeds(ctx context.Context) ([]model.PortSpeed, error)
	SetPortSpeeds(ctx context.Context, port, ingress, egress int) error
	GetCableDiagnostics(ctx context.Context) ([]model.CableDiagnostics, error)
}

func NewClient(address string, m mode) (ClientAware, error) {
	switch m {
	case GUI:
		return gui.NewClient(address), nil
	case TLPP:
		return tlpp.NewClient(address), nil
	default:
		return nil, errors.Join(fmt.Errorf("unknown mode %q", m), clientErrors.ErrInvalidRequest)
	}
}
