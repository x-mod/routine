package routine

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignalCode(t *testing.T) {
	c1 := SignalCode(syscall.SIGINT)
	assert.Equal(t, int32(130), c1.Value())
	assert.Equal(t, "Code(130)", c1.String())

	assert.Equal(t, "OK", OK.String())
	assert.Equal(t, "InvalidArgments", InvalidArgments.String())
}
