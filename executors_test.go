package routine

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// --- Guarantee ---

func TestGuaranteeSuccess(t *testing.T) {
	called := false
	exec := Guarantee(ExecutorFunc(func(ctx context.Context) error {
		called = true
		return nil
	}))
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("inner executor not called")
	}
}

func TestGuaranteeRecoversPanic(t *testing.T) {
	exec := Guarantee(ExecutorFunc(func(ctx context.Context) error {
		panic("boom")
	}))
	// Guarantee swallows the panic and returns nil
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestGuaranteeRecoversPanicError(t *testing.T) {
	exec := Guarantee(ExecutorFunc(func(ctx context.Context) error {
		panic(errors.New("panic-error"))
	}))
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatalf("expected nil (swallowed), got %v", err)
	}
}

func TestGuaranteeInnerError(t *testing.T) {
	exec := Guarantee(ExecutorFunc(func(ctx context.Context) error {
		return errors.New("inner error")
	}))
	// Guarantee logs but returns nil
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

// --- Retry ---

func TestRetrySuccess(t *testing.T) {
	var calls int
	exec := Retry(3, ExecutorFunc(func(ctx context.Context) error {
		calls++
		return nil
	}))
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestRetryExhausted(t *testing.T) {
	want := errors.New("fail")
	exec := Retry(3, ExecutorFunc(func(ctx context.Context) error {
		return want
	}))
	if err := exec.Execute(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestRetrySucceedsOnSecondAttempt(t *testing.T) {
	var calls int
	exec := Retry(3, ExecutorFunc(func(ctx context.Context) error {
		calls++
		if calls < 2 {
			return errors.New("not yet")
		}
		return nil
	}))
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestFromRetry(t *testing.T) {
	var seen []int
	exec := Retry(3, ExecutorFunc(func(ctx context.Context) error {
		seen = append(seen, FromRetry(ctx))
		return errors.New("keep retrying")
	}))
	exec.Execute(context.Background()) //nolint
	if len(seen) != 3 || seen[0] != 1 || seen[1] != 2 || seen[2] != 3 {
		t.Fatalf("unexpected retry counts: %v", seen)
	}
}

func TestFromRetryNoContext(t *testing.T) {
	if got := FromRetry(nil); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
	if got := FromRetry(context.Background()); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

// --- Repeat ---

func TestRepeatFixed(t *testing.T) {
	var calls int32
	exec := Repeat(3, 0, ExecutorFunc(func(ctx context.Context) error {
		atomic.AddInt32(&calls, 1)
		return nil
	}))
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestRepeatError(t *testing.T) {
	exec := Repeat(5, 0, ExecutorFunc(func(ctx context.Context) error {
		return errors.New("fail")
	}))
	if err := exec.Execute(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestFromRepeat(t *testing.T) {
	var seen []int
	exec := Repeat(3, 0, ExecutorFunc(func(ctx context.Context) error {
		seen = append(seen, FromRepeat(ctx))
		return nil
	}))
	exec.Execute(context.Background()) //nolint
	if len(seen) != 3 || seen[0] != 1 || seen[1] != 2 || seen[2] != 3 {
		t.Fatalf("unexpected repeat counts: %v", seen)
	}
}

func TestFromRepeatNoContext(t *testing.T) {
	if got := FromRepeat(nil); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestRepeatWithInterval(t *testing.T) {
	var calls int32
	exec := Repeat(2, 10*time.Millisecond, ExecutorFunc(func(ctx context.Context) error {
		atomic.AddInt32(&calls, 1)
		return nil
	}))
	start := time.Now()
	exec.Execute(context.Background()) //nolint
	elapsed := time.Since(start)
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
	// Should have waited at least one interval (10ms)
	if elapsed < 10*time.Millisecond {
		t.Fatalf("expected at least 10ms elapsed, got %v", elapsed)
	}
}

// --- Crontab ---

func TestFromCrontabNoContext(t *testing.T) {
	if got := FromCrontab(nil); !got.IsZero() {
		t.Fatal("expected zero time")
	}
	if got := FromCrontab(context.Background()); !got.IsZero() {
		t.Fatal("expected zero time")
	}
}

func TestCrontabInvalidPlan(t *testing.T) {
	exec := Crontab("invalid-cron", ExecutorFunc(func(ctx context.Context) error { return nil }))
	if err := exec.Execute(context.Background()); err == nil {
		t.Fatal("expected error for invalid cron expression")
	}
}

func TestCrontabContextCancel(t *testing.T) {
	exec := Crontab("* * * * * * *", ExecutorFunc(func(ctx context.Context) error { return nil }))
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := exec.Execute(ctx)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestCrontabIsTimeMuted(t *testing.T) {
	exec := Crontab("* * * * *", ExecutorFunc(func(ctx context.Context) error { return nil }))

	now := time.Now()
	exec.Mute(now.Add(-time.Hour), now.Add(time.Hour))
	if !exec.IsTimeMuted(now) {
		t.Fatal("expected time to be muted")
	}

	exec2 := Crontab("* * * * *", ExecutorFunc(func(ctx context.Context) error { return nil }))
	exec2.Workday(true) // mute weekends, only fire on workdays
	weekday := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC) // Tuesday
	weekend := time.Date(2024, 1, 6, 12, 0, 0, 0, time.UTC) // Saturday
	if !exec2.IsTimeMuted(weekend) {
		t.Fatal("weekend should be muted when Workday(true)")
	}
	if exec2.IsTimeMuted(weekday) {
		t.Fatal("weekday should not be muted when Workday(true)")
	}

	exec3 := Crontab("* * * * *", ExecutorFunc(func(ctx context.Context) error { return nil }))
	exec3.Weekend(true) // mute weekdays, only fire on weekends
	if !exec3.IsTimeMuted(weekday) {
		t.Fatal("weekday should be muted when Weekend(true)")
	}
	if exec3.IsTimeMuted(weekend) {
		t.Fatal("weekend should not be muted when Weekend(true)")
	}
}

func TestCrontabEveryday(t *testing.T) {
	exec := Crontab("* * * * *", ExecutorFunc(func(ctx context.Context) error { return nil }))
	exec.Workday(true)
	exec.Everyday(true)
	weekday := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)
	weekend := time.Date(2024, 1, 6, 12, 0, 0, 0, time.UTC)
	// Everyday(true) clears both flags
	if exec.IsTimeMuted(weekday) || exec.IsTimeMuted(weekend) {
		t.Fatal("expected no mute after Everyday(true)")
	}
}

// --- Command ---

func TestCommandEcho(t *testing.T) {
	var buf bytes.Buffer
	exec := Command("echo", ARG("hello"), Stdout(&buf))
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "hello") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

func TestCommandNotFound(t *testing.T) {
	exec := Command("this-command-does-not-exist-at-all")
	if err := exec.Execute(context.Background()); err == nil {
		t.Fatal("expected error for missing command")
	}
}

func TestCommandEnv(t *testing.T) {
	var buf bytes.Buffer
	exec := Command("sh", ARG("-c"), ARG("echo $TEST_VAR"), ENV("TEST_VAR=hello_env"), Stdout(&buf))
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "hello_env") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

func TestCommandStdin(t *testing.T) {
	var buf bytes.Buffer
	exec := Command("cat", Stdin(strings.NewReader("from-stdin")), Stdout(&buf))
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "from-stdin" {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

func TestCommandContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	exec := Command("sleep", ARG("10"))
	if err := exec.Execute(ctx); err == nil {
		t.Fatal("expected error from context cancellation")
	}
}

// --- Timeout ---

func TestTimeoutSuccess(t *testing.T) {
	exec := Timeout(time.Second, ExecutorFunc(func(ctx context.Context) error { return nil }))
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestTimeoutExpires(t *testing.T) {
	exec := Timeout(50*time.Millisecond, ExecutorFunc(func(ctx context.Context) error {
		time.Sleep(time.Second)
		return nil
	}))
	if err := exec.Execute(context.Background()); err == nil {
		t.Fatal("expected timeout error")
	}
}

// --- Deadline ---

func TestDeadlineSuccess(t *testing.T) {
	exec := Deadline(time.Now().Add(time.Second), ExecutorFunc(func(ctx context.Context) error { return nil }))
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestDeadlineExpired(t *testing.T) {
	exec := Deadline(time.Now().Add(50*time.Millisecond), ExecutorFunc(func(ctx context.Context) error {
		time.Sleep(time.Second)
		return nil
	}))
	if err := exec.Execute(context.Background()); err == nil {
		t.Fatal("expected deadline error")
	}
}

// --- Concurrent ---

func TestConcurrent(t *testing.T) {
	var calls int32
	exec := Concurrent(5, ExecutorFunc(func(ctx context.Context) error {
		atomic.AddInt32(&calls, 1)
		return nil
	}))
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&calls) != 5 {
		t.Fatalf("expected 5 calls, got %d", calls)
	}
}

func TestConcurrentWithError(t *testing.T) {
	// ConcurrentExecutor logs errors but always returns nil
	exec := Concurrent(3, ExecutorFunc(func(ctx context.Context) error {
		return errors.New("concurrent fail")
	}))
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatalf("expected nil (errors logged), got %v", err)
	}
}

// --- Parallel ---

func TestParallel(t *testing.T) {
	var calls int32
	inc := ExecutorFunc(func(ctx context.Context) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})
	exec := Parallel(inc, inc, inc)
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestParallelWithError(t *testing.T) {
	// ParallelExecutor logs errors but returns nil
	fail := ExecutorFunc(func(ctx context.Context) error { return errors.New("parallel fail") })
	exec := Parallel(fail, fail)
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatalf("expected nil (errors logged), got %v", err)
	}
}

// --- Append ---

func TestAppendSequential(t *testing.T) {
	var order []int
	exec := Append(
		ExecutorFunc(func(ctx context.Context) error { order = append(order, 1); return nil }),
		ExecutorFunc(func(ctx context.Context) error { order = append(order, 2); return nil }),
		ExecutorFunc(func(ctx context.Context) error { order = append(order, 3); return nil }),
	)
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Fatalf("unexpected order: %v", order)
	}
}

func TestAppendStopsOnError(t *testing.T) {
	var calls int
	exec := Append(
		ExecutorFunc(func(ctx context.Context) error { calls++; return nil }),
		ExecutorFunc(func(ctx context.Context) error { calls++; return errors.New("stop") }),
		ExecutorFunc(func(ctx context.Context) error { calls++; return nil }),
	)
	if err := exec.Execute(context.Background()); err == nil {
		t.Fatal("expected error")
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls before stop, got %d", calls)
	}
}

func TestAppendDynamic(t *testing.T) {
	exec := Append()
	var calls int
	exec.Append(ExecutorFunc(func(ctx context.Context) error { calls++; return nil }))
	exec.Append(ExecutorFunc(func(ctx context.Context) error { calls++; return nil }))
	exec.Execute(context.Background()) //nolint
	if calls != 2 {
		t.Fatalf("expected 2, got %d", calls)
	}
}
