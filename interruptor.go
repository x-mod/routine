package routine

import (
	"context"
	"os"
	"syscall"
)

// InterruptHandler definition
type InterruptHandler func(ctx context.Context, cancel context.CancelFunc) (exit bool)

// Interruptor definition
type Interruptor interface {
	Signal() os.Signal
	Interrupt() InterruptHandler
}

//DefaultCancelInterruptors include INT/TERM/KILL signals
var DefaultCancelInterruptors []Interruptor

// CancelInterruptor definition
type CancelInterruptor struct {
	sig os.Signal
	fn  InterruptHandler
}

// NewCancelInterruptor if fn is nil will cancel context
func NewCancelInterruptor(sig os.Signal, fn InterruptHandler) *CancelInterruptor {
	return &CancelInterruptor{
		sig: sig,
		fn:  fn,
	}
}

// Signal inplement the interface
func (c *CancelInterruptor) Signal() os.Signal {
	return c.sig
}

// Interrupt inplement the interface
func (c *CancelInterruptor) Interrupt() InterruptHandler {
	return func(ctx context.Context, cancel context.CancelFunc) bool {
		cancel()
		if c.fn != nil {
			return c.fn(ctx, cancel)
		}
		return true
	}
}

func init() {
	DefaultCancelInterruptors = []Interruptor{
		NewCancelInterruptor(syscall.SIGINT, nil),
		NewCancelInterruptor(syscall.SIGTERM, nil),
		NewCancelInterruptor(syscall.SIGKILL, nil),
	}
}
