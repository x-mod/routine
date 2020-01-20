package main

import (
	"context"
	"log"

	"github.com/x-mod/routine"
)

func main() {
	if err := routine.Main(
		context.TODO(),
		routine.Command("echo", routine.ARG("hello routine!")),
	); err != nil {
		log.Fatal(err)
	}
}
