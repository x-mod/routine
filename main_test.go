package routine

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/x-mod/errors"
)

func TestRun(t *testing.T) {
	err := Main(context.TODO(), ExecutorFunc(func(ctx context.Context) error {
		log.Println("main executing begin ...")

		ch1 := Go(ctx, Retry(3, ExecutorFunc(func(arg1 context.Context) error {
			log.Println("Go1 retry begin ...", FromRetry(arg1))
			time.Sleep(1 * time.Second)
			log.Println("Go1 retry end")
			return errors.New("Go1 error")
		})))
		log.Println("Go1 result: ", <-ch1)

		ch2 := Go(ctx, Repeat(2, time.Second, ExecutorFunc(func(arg1 context.Context) error {
			log.Println("Go2 repeat begin ...", FromRepeat(arg1))
			time.Sleep(2 * time.Second)
			log.Println("Go2 repeat end")
			return nil
		})))
		log.Println("Go2 result: ", <-ch2)

		Go(ctx, Repeat(2, time.Second, Guarantee(ExecutorFunc(func(arg1 context.Context) error {
			log.Println("Go4 repeat guarantee begin ...")
			log.Println("Go4 repeat guarantee end")
			return errors.New("Go4 failed")
		}))))

		Go(ctx, Crontab("* * * * *", ExecutorFunc(func(arg1 context.Context) error {
			log.Println("Go3 crontab begin ...", FromCrontab(arg1))
			log.Println("Go3 crontab end")
			return nil
		})))

		ch5 := Go(ctx, Repeat(3, time.Second, Command("echo", "hello", "routine")))
		log.Println("Go5 result: ", <-ch5)

		ch6 := Go(ctx, Timeout(3*time.Second, Command("sleep", "6")))
		log.Println("Go6 timeout result: ", <-ch6)

		ch7 := Go(ctx, Deadline(time.Now().Add(time.Second), Command("sleep", "6")))
		log.Println("Go7 deadline result: ", <-ch7)

		Go(ctx, Concurrent(20, Command("echo", "hello")))

		log.Println("main executing end")
		return nil
	}), DefaultCancelInterruptors...)
	log.Println("main exit: ", err)
}
