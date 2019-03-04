package routine

import (
	"context"
	"sync"
)

type _wait struct{}

// WithWait context with sync.WaitGroup
func WithWait(ctx context.Context) context.Context {
	if ctx != nil {
		return context.WithValue(ctx, _wait{}, &sync.WaitGroup{})
	}
	return context.WithValue(context.TODO(), _wait{}, &sync.WaitGroup{})
}

// WaitAdd if context with sync.WaitGroup, wait.Add
func WaitAdd(ctx context.Context, delta int) {
	if ctx != nil {
		wait := ctx.Value(_wait{}).(*sync.WaitGroup)
		if wait != nil {
			wait.Add(delta)
		}
	}
}

// WaitDone if context with sync.WaitGroup, wait.Done
func WaitDone(ctx context.Context) {
	if ctx != nil {
		wait := ctx.Value(_wait{}).(*sync.WaitGroup)
		if wait != nil {
			wait.Done()
		}
	}
}

// Wait if context with sync.WaitGroup, wait.Wait for all undone
func Wait(ctx context.Context) {
	if ctx != nil {
		wait := ctx.Value(_wait{}).(*sync.WaitGroup)
		if wait != nil {
			wait.Wait()
		}
	}
}
