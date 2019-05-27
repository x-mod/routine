package routine

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/x-mod/errors"
)

func TestMain(t *testing.T) {
	timeoutCtx, _ := context.WithTimeout(context.TODO(), time.Millisecond*50)
	i := 0
	type args struct {
		exec Executor
		opts []Opt
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"ok",
			args{
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
				ExecutorFunc(func(ctx context.Context) error {
					return errors.New("fail")
				}),
				nil,
			},
			true,
		},
		{
			"argments",
			args{
				ExecutorFunc(func(ctx context.Context) error {
					args, ok := ArgumentsFrom(ctx)
					if !ok {
						return errors.New("argments 1")
					}
					if len(args) != 3 {
						return errors.New("argments 2")
					}
					return nil
				}),
				[]Opt{Arguments(1, "ok", false)},
			},
			false,
		},
		{
			"timeout",
			args{
				ExecutorFunc(func(ctx context.Context) error {
					time.Sleep(time.Second)
					return nil
				}),
				[]Opt{Context(timeoutCtx)},
			},
			true,
		},
		{
			"prepare",
			args{
				ExecutorFunc(func(ctx context.Context) error {
					return nil
				}),
				[]Opt{
					Prepare(ExecutorFunc(func(ctx context.Context) error {
						i = i + 1
						return nil
					})),
					Cleanup(ExecutorFunc(func(ctx context.Context) error {
						if i != 1 {
							return errors.New("prepare failed")
						}
						return nil
					})),
				},
			},
			false,
		},
		{
			"prepare failed",
			args{
				ExecutorFunc(func(ctx context.Context) error {
					return nil
				}),
				[]Opt{
					Prepare(ExecutorFunc(func(ctx context.Context) error {
						return errors.New("prepare failed")
					})),
				},
			},
			true,
		},
		{
			"cleanup failed",
			args{
				ExecutorFunc(func(ctx context.Context) error {
					return nil
				}),
				[]Opt{
					Cleanup(ExecutorFunc(func(ctx context.Context) error {
						return errors.New("cleanup failed")
					})),
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Main(tt.args.exec, tt.args.opts...); (err != nil) != tt.wantErr {
				t.Errorf("Main() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGo(t *testing.T) {
	f1 := func(context.Context) error {
		log.Println("f1 start ...")
		time.Sleep(time.Millisecond * 50)
		log.Println("f1 end.")
		return nil
	}
	assert.Nil(t, nil, Main(ExecutorFunc(f1)))

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
	assert.Nil(t, nil, Main(ExecutorFunc(f3)))

	f4 := func(ctx context.Context) error {
		//wait f1 end
		<-Go(ctx, ExecutorFunc(f1))
		Go(ctx, ExecutorFunc(f2))
		return nil
	}
	assert.Nil(t, nil, Main(ExecutorFunc(f4)))
}

func TestMiddleware(t *testing.T) {
	m1 := func(exec Executor) Executor {
		return ExecutorFunc(func(ctx context.Context) error {
			timeoutCtx, c := context.WithTimeout(ctx, time.Millisecond*50)
			defer c()
			return <-Go(timeoutCtx, exec)
		})
	}

	exe := ExecutorFunc(func(ctx context.Context) error {
		time.Sleep(time.Second)
		return nil
	})

	assert.NotNil(t, <-Go(context.TODO(), m1(exe)))
	assert.NotNil(t, <-Go(context.TODO(), UseExecutorMiddleware(exe, m1)))
}

func TestLog(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.TraceLevel)
	ctx := WithLogger(context.TODO(), log)
	Error(ctx, "error xxx")
	Info(ctx, "info xxx")
	Trace(ctx, "trace xxx")
	Debug(ctx, "debug xxx")
	Warn(ctx, "debug xxx")
}
