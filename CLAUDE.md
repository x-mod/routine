# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build ./...

# Test with coverage
go test -v -coverprofile=coverage.out ./...

# Run a single test
go test -v -run TestName ./...

# Run the quickstart example
go run quickstart/main.go

# Run the full example (generates trace.out for go tool trace)
go run example/main.go

# Inspect traces
go tool trace trace.out
```

## Architecture

This is a Go library (`github.com/x-mod/routine`) for structured goroutine lifecycle management. The core abstraction is the `Executor` interface:

```go
type Executor interface {
    Execute(context.Context) error
}
```

**Core files:**

- `interfaces.go` — `Executor` interface, `ExecutorFunc` adapter, `ExecutorMiddleware` chaining
- `routine.go` — `Routine` struct: the main lifecycle controller. Manages prepare → main execution → child goroutines → cleanup, with signal trapping and `runtime/trace` instrumentation. `New(exec, opts...)` creates a routine; `Execute(ctx)` runs it.
- `main.go` — Package-level `Main(ctx, exec, opts...)` convenience wrapper that stores a global `*Routine`. `Child(ctx, exec)` spawns dynamic child goroutines on the global routine.
- `executors.go` — Composable executor adapters: `Guarantee` (panic recovery), `Retry`, `Repeat`, `Crontab`, `Timeout`, `Deadline`, `Concurrent`, `Parallel`, `Append` (sequential), `Command` (os/exec), `Profiling` (pprof HTTP server)
- `context.go` — Context helpers: `WithArgments`/`ArgumentsFrom` for passing arguments through context; `WithParent`/`ParentFrom` for accessing the parent `*Routine` from context

**Lifecycle model** (`Routine.Execute`):

1. `prepare` executor runs synchronously (optional)
2. Main executor runs in a goroutine, returns on channel
3. Child executors (`Go(...)` opts) run concurrently in goroutines
4. Signal traps start if configured
5. `serving` event fires — routine is now live
6. Blocks until: main executor finishes, context cancels, or `Close()` is called
7. `cleanup` executor runs after all goroutines drain (via `sync.WaitGroup`)

**Executor composition pattern:** All executor adapters wrap an inner `Executor`, enabling arbitrary nesting, e.g., `Retry(3, Timeout(time.Minute, Command("curl", ARG("..."))))`.

**Context values:** Several executors inject metadata into context for introspection by the wrapped executor: `FromRetry(ctx)`, `FromRepeat(ctx)`, `FromCrontab(ctx)`.

## Key dependencies

- `github.com/gorhill/cronexpr` — cron expression parsing for `CrontabExecutor`
- `github.com/x-mod/errors` — error annotation/wrapping
- `github.com/x-mod/event` — one-shot event primitives (`serving`/`stopped` signals)
- `github.com/x-mod/sigtrap` — OS signal capture and handler dispatch
