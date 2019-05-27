routine
===
[![GoDoc](https://godoc.org/github.com/x-mod/routine?status.svg)](https://godoc.org/github.com/x-mod/routine) [![Go Report Card](https://goreportcard.com/badge/github.com/x-mod/routine)](https://goreportcard.com/report/github.com/x-mod/routine) [![Build Status](https://travis-ci.org/x-mod/routine.svg?branch=master)](https://travis-ci.org/x-mod/routine) [![Version](https://img.shields.io/github/tag/x-mod/routine.svg)](https://github.com/x-mod/routine/releases) [![Coverage Status](https://coveralls.io/repos/github/x-mod/routine/badge.svg?branch=master)](https://coveralls.io/github/x-mod/routine?branch=master)

go routine control with context, support: Main, Go, Pool and some useful Executors.

## Why we need control go routine?

The keyword `go` will create a go routine for the function, but if you want to control the routine, it need more work to do, like:

- manage the go routine object
- signal to the running go routine to stop 
- waiting for signals when go routine died

these works are really boring, that's why we need control go routine.

## How to control go routine?

use an `Executor` interface, which user `context.Context` to control the go routine. According to the go routines level, you can use the following entry function help you to control the go routine.

- **Main**, encapsulates default signal handlers, process level waiting for go routines, and with prepare & cleanup options
- **Go**, wrapper for the `go`. You should use it in the `Main` scope.	
- **Pool**, the simplest go routine pool which implement `Routine` interface

## Quick Start

## `Main` function

the `Main` function encapsulates default signal handlers, process level waiting for go routines, and with prepare & cleanup options

````go

import "github.com/x-mod/routine"

func main() {
	err := routine.Main(routine.ExecutorFunc(func(ctx context.Context) error {
		//TODO your code here
		return nil
	}))
	//...
}

````

## `Go` function

the `Go` function is the wrapper of the golang's keyword `go`， when you use the `Go` function, it act the same like keywork `go`, but with inside context controling.

````go

import "github.com/x-mod/routine"

func main() {
	err := routine.Main(routine.ExecutorFunc(func(ctx context.Context) error {
		//ignore the result error
		routine.Go(ctx, routine.ExecutorFunc(func(ctx context.Context) error {
			//go routine 1 ...
			return nil
		}))

		//get the result error
		err := <-routine.Go(ctx, routine.ExecutorFunc(func(ctx context.Context) error {
			//go routine 2 ...
			return nil
		}))
		return nil
	}))
	//...
}

````

## `Pool` routines

````go

import "github.com/x-mod/routine"

func main() {
	//create a pool
	pool := routine.NewPool(routine.NumOfRoutines(4))
	defer pool.Close()
	
	err := routine.Main(routine.ExecutorFunc(func(ctx context.Context) error {	
		//ignore the result error
		pool.Go(ctx, routine.ExecutorFunc(func(ctx context.Context) error {
			//go routine 1 ...
			return nil
		}))

		//get the result error
		err := <-pool.Go(ctx, routine.ExecutorFunc(func(ctx context.Context) error {
			//go routine 2 ...
			return nil
		}))
		return nil
	}))
	//...
}
````

## Executors

provide some useful executors, like:

- retry, retry your executor when failed
- repeat, repeat your executor
- crontab, schedule your executor
- guarantee, make sure your executor never panic

and so on.

````go

import "github.com/x-mod/routine"

func main() {
	
	err := routine.Main(routine.ExecutorFunc(func(ctx context.Context) error {	
		//retry
		routine.Go(ctx, routine.Retry(3, routine.ExecutorFunc(func(ctx context.Context) error {
			//go routine 1 ...
			return nil
		})))

		//guarantee
		routine.Go(ctx, routine.Guarantee(3, routine.ExecutorFunc(func(ctx context.Context) error {
			panic("panic")
			return nil
		})))

		//concurrent
		routine.Go(ctx, routine.Concurrent(10, routine.ExecutorFunc(func(ctx context.Context) error {
			return nil
		})))
	
		//timeout
		routine.Go(ctx, routine.Timeout(time.Second, routine.ExecutorFunc(func(ctx context.Context) error {
			return nil
		})))
		return nil
	}))
	//...
}
````