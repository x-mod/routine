package routine

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/x-mod/errors"
)

//Main wrapper for executor with waits & signal interuptors
func Main(exec Executor, opts ...Opt) error {
	moptions := &options{
		ctx: context.TODO(),
	}
	for _, opt := range opts {
		opt(moptions)
	}
	parent := moptions.ctx
	//routine pool
	if moptions.pool != nil {
		parent = WithRoutine(parent, moptions.pool)
	}
	//prepare
	if moptions.prepareExec != nil {
		if err := moptions.prepareExec.Execute(parent); err != nil {
			return errors.Annotate(err, "routine main prepare")
		}
	}
	// context with cancel & wait
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	// argments
	if len(moptions.args) > 0 {
		ctx = WithArgments(ctx, moptions.args...)
	}
	// main ctx with wait
	ctx = WithWait(ctx)
	// signals
	sigchan := make(chan os.Signal)
	sighandlers := make(map[os.Signal]InterruptHandler)
	for _, interruptor := range moptions.interrupts {
		signal.Notify(sigchan, interruptor.Signal())
		sighandlers[interruptor.Signal()] = interruptor.Interrupt()
	}
	// main executor
	ch := Go(ctx, exec)
	// main exit for sig & finished
	exitCode := GeneralErr
	exitCh := make(chan error, 1)
	for {
		select {
		case sig := <-sigchan:
			// cancel when a signal catched
			if h, ok := sighandlers[sig]; ok {
				if h(ctx) {
					cancel()
					exitCh <- errors.CodeError(SignalCode(sig.(syscall.Signal)))
					goto Exit
				}
			}
		case <-ctx.Done():
			exitCh <- errors.WithCode(ctx.Err(), exitCode)
			goto Exit
		case err := <-ch:
			exitCh <- err
			goto Exit
		}
	}
Exit:
	//close running routines
	if moptions.pool != nil {
		moptions.pool.Close()
	}
	//cleanup
	if moptions.cleanupExec != nil {
		if err := moptions.cleanupExec.Execute(parent); err != nil {
			return errors.Annotate(err, "routine main cleanup")
		}
	}
	//wait main context subroutines
	Wait(ctx)
	return <-exitCh
}

//Go wrapper for go keyword, use in MAIN function
func Go(ctx context.Context, exec Executor) chan error {
	//chan capcity set 2 for noneblock
	ch := make(chan error, 2)
	if exec == nil {
		ch <- ErrNoneExecutor
		return ch
	}
	if ctx == nil {
		ch <- ErrNoneContext
		return ch
	}
	//use context routine pool
	if rs, ok := RoutineFrom(ctx); ok {
		return rs.Go(ctx, exec)
	}
	WaitAdd(ctx, 2)
	go func() {
		defer WaitDone(ctx)
		// channel for function (run) done
		stop := make(chan struct{})
		go func() {
			defer WaitDone(ctx)
			ch <- exec.Execute(ctx)
			close(stop)
		}()
		// run exit for cancel & finished
		select {
		case <-ctx.Done():
			ch <- ctx.Err()
		case <-stop:
		}
	}()
	return ch
}

type options struct {
	ctx         context.Context
	args        []interface{}
	interrupts  []Interruptor
	pool        *Pool
	prepareExec Executor
	cleanupExec Executor
}

//Opt interface
type Opt func(*options)

//Context Opt
func Context(ctx context.Context) Opt {
	return func(opts *options) {
		if ctx != nil {
			opts.ctx = ctx
		}
	}
}

//Arguments Opt for Main
func Arguments(args ...interface{}) Opt {
	return func(opts *options) {
		opts.args = args
	}
}

//Interrupts Opt for Main
func Interrupts(ints ...Interruptor) Opt {
	return func(opts *options) {
		opts.interrupts = append(opts.interrupts, ints...)
	}
}

//WithPool Opt for Main
func WithPool(p *Pool) Opt {
	return func(opts *options) {
		opts.pool = p
	}
}

//Prepare Opt for Main
func Prepare(exec Executor) Opt {
	return func(opts *options) {
		opts.prepareExec = exec
	}
}

//Cleanup Opt for Main
func Cleanup(exec Executor) Opt {
	return func(opts *options) {
		opts.cleanupExec = exec
	}
}
