package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/x-mod/routine"
)

func foo(context.Context) error {
	log.Println("foo begin")
	time.Sleep(time.Second * 2)
	log.Println("foo end")
	return errors.New("foo done")
}

func main() {
	tmctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()
	err := routine.Main(tmctx,
		routine.Concurrent(4, routine.ExecutorFunc(foo)),
		routine.Cleanup(routine.ExecutorFunc(func(ctx context.Context) error {
			log.Println("clear")
			return nil
		})),
	)
	if err != nil {
		log.Println("failed:", err)
	}
}
