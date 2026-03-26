package client_test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"

	client "github.com/robotjoosen/go-tplink-easysmart-client"
)

func Test_Client(t *testing.T) {
	c := client.NewClient("192.168.2.101", client.TLPP)

	assert.NoError(t, c.Login(context.Background(), "admin", "REDACTED"))
}
