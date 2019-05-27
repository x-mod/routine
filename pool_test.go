package routine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/x-mod/errors"
)

func TestPool(t *testing.T) {
	p1 := NewPool()
	defer p1.Close()
	assert.NotNil(t, p1)
	ch1 := p1.Go(context.TODO(), ExecutorFunc(func(ctx context.Context) error {
		return nil
	}))
	assert.Nil(t, <-ch1)

	ch2 := p1.Go(context.TODO(), ExecutorFunc(func(ctx context.Context) error {
		return errors.New("failed")
	}))
	assert.NotNil(t, <-ch2)

	p2 := NewPool(NumOfRoutines(4), MaxOfRequestBufferSize(16))
	chs := make([]chan error, 0, 20)
	for i := 0; i < 20; i++ {
		ch := p2.Go(context.TODO(), ExecutorFunc(func(ctx context.Context) error {
			return nil
		}))
		chs = append(chs, ch)
	}
	for i := 0; i < 20; i++ {
		assert.Nil(t, <-chs[i])
	}

	p3 := NewPool()
	defer p3.Close()
	timeoutCtx, c := context.WithTimeout(context.TODO(), time.Second)
	defer c()
	ch3 := p3.Go(timeoutCtx, ExecutorFunc(func(ctx context.Context) error {
		time.Sleep(2 * time.Second)
		return nil
	}))
	err := <-ch3
	assert.Nil(t, err, err)

	p4 := NewPool()
	defer p4.Close()
	timeoutCtx1, c1 := context.WithTimeout(context.TODO(), time.Millisecond*500)
	defer c1()
	time.Sleep(time.Second)
	ch4 := p4.Go(timeoutCtx1, ExecutorFunc(func(ctx context.Context) error {
		return nil
	}))
	assert.NotNil(t, <-ch4)
}
