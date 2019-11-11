package routine

import "context"

func Main(ctx context.Context, exec Executor, opts ...Opt) error {
	return New(exec, opts...).Execute(ctx)
}
