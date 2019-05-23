package routine

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/x-mod/errors"
)

func TestMain(t *testing.T) {
	type args struct {
		parent context.Context
		exec   Executor
		opts   []Opt
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"ok",
			args{
				context.TODO(),
				ExecutorFunc(func(ctx context.Context) error {
					return nil
				}),
				nil,
			},
			false,
		},
		{
			"fail",
			args{
				context.TODO(),
				ExecutorFunc(func(ctx context.Context) error {
					return errors.New("fail")
				}),
				nil,
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Main(tt.args.parent, tt.args.exec, tt.args.opts...); (err != nil) != tt.wantErr {
				t.Errorf("Main() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGo(t *testing.T) {
	f1 := func(context.Context) error {
		log.Println("f1 start ...")
		time.Sleep(time.Second)
		log.Println("f1 end.")
		return nil
	}
	assert.Nil(t, nil, Main(context.TODO(), ExecutorFunc(f1)))

	f2 := func(context.Context) error {
		log.Println("f2 start ...")
		log.Println("f2 end.")
		return nil
	}
	f3 := func(ctx context.Context) error {
		Go(ctx, ExecutorFunc(f1))
		Go(ctx, ExecutorFunc(f2))
		return nil
	}
	assert.Nil(t, nil, Main(context.TODO(), ExecutorFunc(f3)))

	f4 := func(ctx context.Context) error {
		//wait f1 end
		<-Go(ctx, ExecutorFunc(f1))
		Go(ctx, ExecutorFunc(f2))
		return nil
	}
	assert.Nil(t, nil, Main(nil, ExecutorFunc(f4)))
}
