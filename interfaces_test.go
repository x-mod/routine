package routine

import (
	"context"
	"errors"
	"testing"
)

func TestExecutorFunc(t *testing.T) {
	called := false
	f := ExecutorFunc(func(ctx context.Context) error {
		called = true
		return nil
	})
	if err := f.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("executor not called")
	}
}

func TestExecutorFuncError(t *testing.T) {
	want := errors.New("exec error")
	f := ExecutorFunc(func(ctx context.Context) error { return want })
	if got := f.Execute(context.Background()); got != want {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestUseExecutorMiddleware(t *testing.T) {
	order := []int{}
	base := ExecutorFunc(func(ctx context.Context) error {
		order = append(order, 0)
		return nil
	})
	m1 := ExecutorMiddleware(func(next Executor) Executor {
		return ExecutorFunc(func(ctx context.Context) error {
			order = append(order, 1)
			return next.Execute(ctx)
		})
	})
	m2 := ExecutorMiddleware(func(next Executor) Executor {
		return ExecutorFunc(func(ctx context.Context) error {
			order = append(order, 2)
			return next.Execute(ctx)
		})
	})
	exec := UseExecutorMiddleware(base, m1, m2)
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Middleware applied in reverse: m1 wraps m2 wraps base → order 1,2,0
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 0 {
		t.Fatalf("unexpected call order: %v", order)
	}
}

func TestUseExecutorMiddlewareNone(t *testing.T) {
	base := ExecutorFunc(func(ctx context.Context) error { return nil })
	exec := UseExecutorMiddleware(base)
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
}
