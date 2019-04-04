package routine

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/x-mod/errors"
)

//Main wrapper for executor with waits & signal interuptors
func Main(parent context.Context, exec Executor, interruptors ...Interruptor) error {
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

	ch := make(chan error)
	go func() {
		//main executing
		if err := exec.Execute(ctx); err != nil {
			ch <- err
			return
		}

		//time after a short time interval, reschedule to subroutines
		<-time.After(50 * time.Millisecond)

		//waiting for subroutines finishing
		Wait(ctx)
		close(ch)
	}()

	// main exit for sig & finished
	for {
		select {
		case sig := <-sigchan:
			// cancel when a signal catched
			if h, ok := sighandlers[sig]; ok {
				if h(ctx) {
					return errors.CodeError(SignalCode(sig.(syscall.Signal)))
				}
			}
		case <-ctx.Done():
			return errors.WithCode(ctx.Err(), GeneralErr)
		case err, ok := <-ch:
			if !ok {
				return nil
			}
			return errors.WithCode(err, GeneralErr)
		}
	}
}

//Go wrapper for go keyword, use in MAIN function
func Go(ctx context.Context, exec Executor) chan error {
	ch := make(chan error, 1)
	if exec == nil {
		ch <- ErrNoneExecutor
		return ch
	}
	if ctx == nil {
		ch <- ErrNoneContext
		return ch
	}

	go func() {
		WaitAdd(ctx, 1)
		defer WaitDone(ctx)

		// channel for function (run) done
		stop := make(chan struct{})
		go func() {
			ch <- exec.Execute(ctx)
			close(stop)
		}()

		runtime.Gosched()
		// run exit for cancel & finished
		select {
		case <-ctx.Done():
			ch <- ctx.Err()
		case <-stop:
		}
	}()
	return ch
}
