package client_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	client "github.com/robotjoosen/go-tplink-easysmart-client"
	clientErrors "github.com/robotjoosen/go-tplink-easysmart-client/pkg/errors"
)

func realSwitch(t *testing.T) (addr, password string) {
	t.Helper()

	addr, password = os.Getenv("SWITCH_ADDRESS"), os.Getenv("SWITCH_PASSWORD")
	if addr == "" || password == "" {
		t.Skip("set SWITCH_ADDRESS and SWITCH_PASSWORD to run against a real switch")
	}

	return addr, password
}

// Test_Client_GUI runs read-only calls against a real switch's web interface.
func Test_Client_GUI(t *testing.T) {
	addr, password := realSwitch(t)

	c, err := client.NewClient(addr, client.GUI)
	assert.NoError(t, err)

	ctx := context.Background()
	assert.NoError(t, c.Login(ctx, "admin", password))

	device, err := c.GetDevice(ctx)
	assert.NoError(t, err)
	t.Logf("device: %+v", device)

	ports, err := c.GetPortStats(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, ports)
	t.Logf("ports: %d, first: %+v", len(ports), ports[:min(1, len(ports))])

	speeds, err := c.GetPortSpeeds(ctx)
	assert.NoError(t, err)
	assert.Len(t, speeds, len(ports))
}

func Test_NewClient_UnknownMode(t *testing.T) {
	c, err := client.NewClient("127.0.0.1", "nope")

	assert.Nil(t, c)
	assert.ErrorIs(t, err, clientErrors.ErrInvalidRequest)
}
