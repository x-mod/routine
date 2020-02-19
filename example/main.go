package main

import (
	"context"
	"log"
	"os"
	"syscall"
	"time"

	_ "net/http/pprof"
	"runtime/trace"

	"github.com/x-mod/routine"
)

func prepare(ctx context.Context) error {
	log.Println("prepare begin")
	defer log.Println("prepare end")
	trace.Logf(ctx, "prepare", "prepare ... ok")
	return nil
}

func cleanup(ctx context.Context) error {
	log.Println("cleanup begin")
	defer log.Println("cleanup end")
	time.Sleep(time.Millisecond * 50)
	trace.Logf(ctx, "cleanup", "cleanup ... ok")
	return nil
}

func foo(ctx context.Context) error {
	log.Println("foo begin")
	defer log.Println("foo end")
	time.Sleep(time.Second * 2)
	trace.Logf(ctx, "foo", "sleeping 2s done")
	return nil
}

func bar(ctx context.Context) error {
	log.Println("bar begin")
	defer log.Println("bar end")
	for i := 0; i < 10; i++ {
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
		routine.Signal(syscall.SIGINT, routine.SigHandler(func() {
			os.Exit(1)
		})),
		routine.Prepare(routine.ExecutorFunc(prepare)),
		routine.Cleanup(routine.ExecutorFunc(cleanup)),
		routine.Trace(f),
		routine.Go(routine.ExecutorFunc(foo)),
		routine.Go(routine.ExecutorFunc(foo)),
		routine.Go(routine.ExecutorFunc(foo)),
	); err != nil {
		log.Println(err)
	}
}
