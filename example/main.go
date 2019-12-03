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
	err := routine.Main(
		context.TODO(),
		routine.ExecutorFunc(foo),
	)
	if err != nil {
		log.Fatal(err)
	}
}
