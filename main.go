package routine

import (
	"context"
	"os"
	"os/signal"
	"runtime"
)

type _argments struct{}

//WithArgments inject into context
func WithArgments(ctx context.Context, args ...interface{}) context.Context {
	if ctx != nil {
		return context.WithValue(ctx, _argments{}, args)
	}
	return context.WithValue(context.TODO(), _argments{}, args)
}

//ArgumentsFrom extract from context
func ArgumentsFrom(ctx context.Context) []interface{} {
	if ctx != nil {
		argments := ctx.Value(_argments{})
		if argments != nil {
			return argments.([]interface{})
		}
	}
	return []interface{}{}
}

//Main wrapper for executor with waits & signal interuptors
func Main(parent context.Context, exec Executor, interruptors ...Interruptor) {
	// context with cancel & wait
	ctx, cancel := context.WithCancel(WithWait(parent))
	defer cancel()

	// signals
	sigchan := make(chan os.Signal)
	sighandlers := make(map[os.Signal]InterruptHandler)
	for _, interruptor := range interruptors {
		signal.Notify(sigchan, interruptor.Signal())
		sighandlers[interruptor.Signal()] = interruptor.Interrupt()
	}

	// channel for function (run) done
	if exec != nil {
		ch := make(chan struct{})
		go func(ctx context.Context) {
			argments := ArgumentsFrom(ctx)
			exec.Execute(ctx, argments...)
			ch <- struct{}{}
		}(ctx)

		//yields to other goroutines
		runtime.Gosched()

		// main exit for sig & finished
		for {
			select {
			case sig := <-sigchan:
				// cancel when a signal catched
				if handler, ok := sighandlers[sig]; ok {
					if quit := handler(ctx, cancel); quit {
						goto Quit
					}
				}
			case <-ctx.Done():
				goto Quit
			case <-ch:
				goto Quit
			}
		}
	}

Quit:
	// wait goroutines
	Wait(ctx)
}

func execute(ctx context.Context, exec Executor) {
	if exec == nil {
		return
	}
	if ctx == nil {
		panic("context should never be null")
	}
	WaitAdd(ctx, 1)
	defer WaitDone(ctx)

	// channel for function (run) done
	ch := make(chan struct{})
	go func(ctx context.Context) {
		argments := ArgumentsFrom(ctx)
		exec.Execute(ctx, argments...)
		ch <- struct{}{}
	}(ctx)

	//yields to other goroutines
	runtime.Gosched()
	// run exit for cancel & finished
	select {
	case <-ctx.Done():
	case <-ch:
	}
}

//Go wrapper for system go func
func Go(ctx context.Context, exec Executor) {
	go execute(ctx, exec)
}
