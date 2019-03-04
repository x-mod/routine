package routine

import (
	"context"
	"log"
	"testing"
)

func TestMain(t *testing.T) {
	type args struct {
		parent       context.Context
		exec         Executor
		interruptors []Interruptor
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"main",
			args{
				WithArgments(context.TODO(), "main", 1, false, "sdfds"),
				ExecutorFunc(func(ctx context.Context, args ...interface{}) {
					t.Log(args)
				}),
				[]Interruptor{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Main(tt.args.parent, tt.args.exec, tt.args.interruptors...)
		})
	}
}

func TestRun(t *testing.T) {
	type args struct {
		ctx  context.Context
		exec Executor
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"run",
			args{
				WithArgments(context.TODO(), "run", 1, false, "sdfds"),
				ExecutorFunc(func(ctx context.Context, args ...interface{}) {
					log.Println("runner ...")
					t.Log(args)
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Main(tt.args.ctx, ExecutorFunc(func(ctx context.Context, args ...interface{}) {
				Go(ctx, tt.args.exec)
				Go(ctx, tt.args.exec)
				Go(ctx, tt.args.exec)
			}))
		})
	}
}
