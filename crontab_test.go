package routine

import (
	"context"
	"log"
	"testing"
	"time"
)

func TestCrontab_Open(t *testing.T) {
	Main(nil, ExecutorFunc(func(ctx context.Context, args ...interface{}) {
		crontab := NewCrontab()
		if err := crontab.Open(ctx); err != nil {
			log.Println("crontab open failed:", err)
			return
		}
		defer crontab.Close()

		crontab.JOB("* * * * *", ExecutorFunc(func(ctx context.Context, args ...interface{}) {
			log.Println(args...)
		})).Go(ctx, 1)

		crontab.EVERY(3*time.Second, ExecutorFunc(func(ctx context.Context, args ...interface{}) {
			log.Println(args...)
		})).Go(ctx, 2)

		time.Sleep(time.Minute)
	}))
}
