package routine

import (
	"context"
	"log"
	"testing"
	"time"
)

func TestPool_Go(t *testing.T) {
	Main(nil, ExecutorFunc(func(ctx context.Context, args ...interface{}) {
		pool := NewPool(RunningSize(5), WaitingSize(5))
		if err := pool.Open(ctx); err != nil {
			t.Error(err, "pool1 open")
		}
		defer pool.Close()

		for i := 0; i < 10; i++ {
			pool.Go(WithArgments(ctx, i), ExecutorFunc(func(ctx context.Context, args ...interface{}) {
				log.Println(ArgumentsFrom(ctx)...)
				time.Sleep(2 * time.Second)
			}))
		}

		pool2 := NewPool(FixedExecutor(ExecutorFunc(func(ctx context.Context, args ...interface{}) {
			log.Println(ArgumentsFrom(ctx)...)
		})))
		defer pool2.Close()
		if err := pool2.Open(ctx); err != nil {
			t.Error(err, "pool2 open")
		}

		for i := 0; i < 10; i++ {
			pool2.Execute(ctx, i)
		}

	}))
}

func TestChannel(t *testing.T) {
	c := make(chan int, 10)
	for i := 0; i < 5; i++ {
		c <- i
	}
	close(c)

	for {
		select {
		case v, ok := <-c:
			if !ok {
				log.Println("c closed")
				return
			}
			log.Println("c get ", v)
		}
	}

}
