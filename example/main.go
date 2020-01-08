package main

import (
	"context"
	"log"
	"os"
	"time"

	"runtime/trace"

	"github.com/x-mod/routine"
)

func foo(ctx context.Context) error {
	// trace.Log(ctx, "foo", "starting ...")
	// defer trace.Log(ctx, "foo", "stopped")
	log.Println("foo begin")
	defer log.Println("foo end")
	time.Sleep(time.Second * 2)
	return nil
}

func bar(ctx context.Context) error {
	// trace.Log(ctx, "bar", "starting ...")
	// defer trace.Log(ctx, "bar", "stopped")
	log.Println("bar begin")
	defer log.Println("bar end")
	for i := 0; i < 100; i++ {
		log.Println(i)
		trace.Logf(ctx, "bar", "counting ... %d", i)
	}
	return nil
}

func main() {
	f, err := os.Create("trace.out")
	if err != nil {
		log.Fatalf("failed to create trace output file: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatalf("failed to close trace file: %v", err)
		}
	}()

	if err := routine.Main(
		context.TODO(),
		routine.ExecutorFunc(bar),
		routine.Prepare(routine.ExecutorFunc(foo)),
		routine.Trace(f),
		routine.Go(routine.ExecutorFunc(foo)),
		routine.Go(routine.ExecutorFunc(foo)),
		routine.Go(routine.ExecutorFunc(foo)),
		routine.Interrupts(routine.DefaultCancelInterruptors...),
	); err != nil {
		log.Fatal(err)
	}
}
