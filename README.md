routine
===
[![GoDoc](https://godoc.org/github.com/x-mod/routine?status.svg)](https://godoc.org/github.com/x-mod/routine) [![Go Report Card](https://goreportcard.com/badge/github.com/x-mod/routine)](https://goreportcard.com/report/github.com/x-mod/routine) [![Build Status](https://travis-ci.org/x-mod/routine.svg?branch=master)](https://travis-ci.org/x-mod/routine) [![Version](https://img.shields.io/github/tag/x-mod/routine.svg)](https://github.com/x-mod/routine/releases) [![Coverage Status](https://coveralls.io/repos/github/x-mod/routine/badge.svg?branch=master)](https://coveralls.io/github/x-mod/routine?branch=master)

go routine control with context, support: Main, Go, Pool and some useful Executors.

## Main

````go
package main

import (
	"context"
	"errors"
	"log"
	
	"github.com/x-mod/routine"
)

func foo(ctx context.Context) error {
	log.Println("foo begin")
	log.Println("foo end")
	return errors.New("foo done")
}

func main(){
	err := routine.Main(
		context.TODO(),
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
````

## Routine

````go

import "github.com/x-mod/routine"

err := routine.New(execute).Execute(ctx)

````

## Executors

````go

import "github.com/x-mod/routine"

crontab := routine.Crontab("* * * * *", execute)

````

# Enjoy
