package routine

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	ctx := WithArgments(nil, 1, 2, 3)
	assert.Nil(t, ctx)
	args, ok := ArgumentsFrom(ctx)
	assert.Equal(t, false, ok)
	assert.Nil(t, args)
	ctx = WithArgments(context.TODO(), 1, 2, 3)
	assert.NotNil(t, ctx)
	args, ok = ArgumentsFrom(ctx)
	assert.Equal(t, true, ok)
	assert.Equal(t, 3, len(args))

	ctx1 := WithStdin(nil, os.Stdin)
	assert.NotNil(t, ctx1)
	assert.Equal(t, os.Stdin, StdinFrom(ctx1))
	ctx2 := WithStdout(nil, os.Stdout)
	assert.NotNil(t, ctx2)
	assert.Equal(t, os.Stdout, StdoutFrom(ctx2))
	ctx3 := WithStderr(nil, os.Stderr)
	assert.NotNil(t, ctx3)
	assert.Equal(t, os.Stderr, StderrFrom(ctx3))

	rw := bytes.NewBuffer([]byte{})
	ctx1 = WithStdin(context.TODO(), rw)
	assert.NotNil(t, ctx1)
	assert.Equal(t, rw, StdinFrom(ctx1))
	ctx2 = WithStdout(context.TODO(), rw)
	assert.NotNil(t, ctx2)
	assert.Equal(t, rw, StdoutFrom(ctx2))
	ctx3 = WithStderr(context.TODO(), rw)
	assert.NotNil(t, ctx3)
	assert.Equal(t, rw, StderrFrom(ctx3))

}
