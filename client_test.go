package client_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	client "github.com/robotjoosen/go-tplink-easysmart-client"
)

func Test_Client(t *testing.T) {
	addr, password := os.Getenv("SWITCH_ADDRESS"), os.Getenv("SWITCH_PASSWORD")
	if addr == "" || password == "" {
		t.Skip("set SWITCH_ADDRESS and SWITCH_PASSWORD to run against a real switch")
	}

	c := client.NewClient(addr, client.TLPP)

	assert.NoError(t, c.Login(context.Background(), "admin", password))
}
