package tlpp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	clientErrors "github.com/robotjoosen/go-tplink-easysmart-client/pkg/errors"
)

func Test_GetCableDiagnostics_NotSupported(t *testing.T) {
	c := NewClient("127.0.0.1")

	_, err := c.GetCableDiagnostics(context.Background())

	assert.ErrorIs(t, err, clientErrors.ErrNotSupported)
}
