package routine

import (
	"context"
)

//Executor interface definition
type Executor interface {
	Execute(context.Context, ...interface{})
}

//ExecutorFunc definition
type ExecutorFunc func(context.Context, ...interface{})

//Execute ExecutorFunc implemention of Executor
func (f ExecutorFunc) Execute(ctx context.Context, args ...interface{}) {
	f(ctx, args...)
}

// ExecutorMiddleware is a function that middlewares can implement to be
// able to chain.
type ExecutorMiddleware func(Executor) Executor

// UseExecutorMiddleware wraps a Executor in one or more middleware.
func UseExecutorMiddleware(exec Executor, middleware ...ExecutorMiddleware) Executor {
	// Apply in reverse order.
	for i := len(middleware) - 1; i >= 0; i-- {
		m := middleware[i]
		exec = m(exec)
	}
	return exec
}
