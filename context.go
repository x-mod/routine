package routine

import (
	"context"
)

type _argments struct{}

//WithArgments inject into context
func WithArgments(ctx context.Context, args ...interface{}) context.Context {
	if ctx != nil {
		return context.WithValue(ctx, _argments{}, args)
	}
	return nil
}

//ArgumentsFrom extract from context
func ArgumentsFrom(ctx context.Context) ([]interface{}, bool) {
	if ctx != nil {
		argments := ctx.Value(_argments{})
		if argments != nil {
			return argments.([]interface{}), true
		}
	}
	return nil, false
}

type _parent_routine struct{}

func WithParent(ctx context.Context, parent *Routine) context.Context {
	return context.WithValue(ctx, _parent_routine{}, parent)
}

func ParentFrom(ctx context.Context) (*Routine, bool) {
	parent := ctx.Value(_parent_routine{})
	if parent != nil {
		return parent.(*Routine), true
	}
	return nil, false
}
