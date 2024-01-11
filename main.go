package routine

import "context"

func Main(ctx context.Context, exec Executor, opts ...Opt) error {
	main = New(exec, opts...)
	return main.Execute(ctx)
}

var main *Routine

func Child(ctx context.Context, exec Executor) chan error {
	return main.childGo("appendGo", ctx, exec)
}
