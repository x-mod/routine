package routine

import (
	"context"
	"errors"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func noop() ExecutorFunc {
	return ExecutorFunc(func(ctx context.Context) error { return nil })
}

func blocker() ExecutorFunc {
	return ExecutorFunc(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
}

// --- Basic Routine lifecycle ---

func TestRoutineRunsAndReturns(t *testing.T) {
	r := New(noop())
	if err := r.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestRoutineNilContextError(t *testing.T) {
	r := New(noop())
	if err := r.Execute(nil); err == nil {
		t.Fatal("expected error for nil context")
	}
}

func TestRoutineMainError(t *testing.T) {
	want := errors.New("main fail")
	r := New(ExecutorFunc(func(ctx context.Context) error { return want }))
	if err := r.Execute(context.Background()); err != want {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func TestRoutineContextCancelStops(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	r := New(blocker())
	err := r.Execute(ctx)
	if err == nil {
		t.Fatal("expected context error")
	}
}

// --- Prepare / Cleanup ---

func TestRoutinePrepareRuns(t *testing.T) {
	prepared := false
	r := New(noop(), Prepare(ExecutorFunc(func(ctx context.Context) error {
		prepared = true
		return nil
	})))
	r.Execute(context.Background()) //nolint
	if !prepared {
		t.Fatal("prepare not called")
	}
}

func TestRoutinePrepareError(t *testing.T) {
	want := errors.New("prepare fail")
	r := New(noop(), Prepare(ExecutorFunc(func(ctx context.Context) error { return want })))
	if err := r.Execute(context.Background()); err != want {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func TestRoutineCleanupRuns(t *testing.T) {
	cleaned := false
	r := New(noop(), Cleanup(ExecutorFunc(func(ctx context.Context) error {
		cleaned = true
		return nil
	})))
	r.Execute(context.Background()) //nolint
	if !cleaned {
		t.Fatal("cleanup not called")
	}
}

// --- Child goroutines ---

func TestRoutineChildGoRuns(t *testing.T) {
	childDone := make(chan struct{})
	child := ExecutorFunc(func(ctx context.Context) error {
		close(childDone)
		return nil
	})
	r := New(noop(), Go(child))
	r.Execute(context.Background()) //nolint
	select {
	case <-childDone:
	case <-time.After(time.Second):
		t.Fatal("child never ran")
	}
}

func TestRoutineMultipleChildren(t *testing.T) {
	var count int
	done := make(chan struct{}, 3)
	child := ExecutorFunc(func(ctx context.Context) error {
		done <- struct{}{}
		return nil
	})
	r := New(noop(), Go(child), Go(child), Go(child))
	r.Execute(context.Background()) //nolint
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			count++
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for child")
		}
	}
	if count != 3 {
		t.Fatalf("expected 3 children done, got %d", count)
	}
}

func TestGoOptIgnoresNil(t *testing.T) {
	// Go(nil) should not panic or add a nil child
	opts := &options{}
	Go(nil)(opts)
	if len(opts.childs) != 0 {
		t.Fatal("nil child should not be added")
	}
}

// --- Close ---

func TestRoutineClose(t *testing.T) {
	r := New(blocker())
	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Execute(context.Background())
	}()
	// Wait until serving
	select {
	case <-r.Serving():
	case <-time.After(time.Second):
		t.Fatal("routine never served")
	}
	stoppedCh := r.Close()
	select {
	case <-stoppedCh:
	case <-time.After(time.Second):
		t.Fatal("routine never stopped after Close")
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected nil after close, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Execute never returned")
	}
}

func TestRoutineCloseBeforeServing(t *testing.T) {
	r := New(noop())
	// Close before Execute — should return immediately (event.Done())
	ch := r.Close()
	select {
	case <-ch:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Close before serving should return immediately")
	}
}

// --- Serving event ---

func TestRoutineServingFires(t *testing.T) {
	r := New(blocker())
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		r.Execute(ctx) //nolint
	}()
	select {
	case <-r.Serving():
	case <-time.After(time.Second):
		t.Fatal("serving event never fired")
	}
}

// --- Arguments ---

func TestRoutineArguments(t *testing.T) {
	var got []interface{}
	r := New(
		ExecutorFunc(func(ctx context.Context) error {
			args, ok := ArgumentsFrom(ctx)
			if ok {
				got = args
			}
			return nil
		}),
		Arguments("hello", 42),
	)
	r.Execute(context.Background()) //nolint
	if len(got) != 2 || got[0] != "hello" || got[1] != 42 {
		t.Fatalf("unexpected arguments: %v", got)
	}
}

// --- Signal ---

func TestRoutineCancelSignal(t *testing.T) {
	r := New(blocker(), CancelSignals(syscall.SIGUSR1))
	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Execute(context.Background())
	}()
	select {
	case <-r.Serving():
	case <-time.After(time.Second):
		t.Fatal("routine never served")
	}
	time.Sleep(50 * time.Millisecond) // allow sigtrap goroutine to start listening
	syscall.Kill(syscall.Getpid(), syscall.SIGUSR1) //nolint
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected context error after signal")
		}
	case <-time.After(time.Second):
		t.Fatal("routine did not stop after signal")
	}
}

func TestRoutineSignalHandler(t *testing.T) {
	handled := make(chan struct{}, 1)
	// Use a dedicated signal handler; context timeout stops the routine
	r := New(blocker(), Signal(syscall.SIGUSR2, func() {
		handled <- struct{}{}
	}))
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		r.Execute(ctx) //nolint
	}()
	select {
	case <-r.Serving():
	case <-time.After(time.Second):
		t.Fatal("routine never served")
	}
	time.Sleep(50 * time.Millisecond) // allow sigtrap goroutine to start listening
	syscall.Kill(syscall.Getpid(), syscall.SIGUSR2) //nolint
	select {
	case <-handled:
	case <-time.After(time.Second):
		t.Fatal("signal handler never called")
	}
}

// --- Composition: Timeout wrapping Routine ---

func TestTimeoutWrapsRoutine(t *testing.T) {
	exec := Timeout(50*time.Millisecond, ExecutorFunc(func(ctx context.Context) error {
		time.Sleep(500 * time.Millisecond)
		return nil
	}))
	if err := exec.Execute(context.Background()); err == nil {
		t.Fatal("expected timeout error")
	}
}

// --- Repeat infinite cancelled by context ---

func TestRepeatInfiniteContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	var calls int32
	// repeat=0 means infinite
	exec := Repeat(0, 10*time.Millisecond, ExecutorFunc(func(ctx context.Context) error {
		// check context each iteration to avoid blocking
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		atomic.AddInt32(&calls, 1)
		return nil
	}))
	exec.Execute(ctx) //nolint
	if atomic.LoadInt32(&calls) == 0 {
		t.Fatal("expected at least one iteration")
	}
}
