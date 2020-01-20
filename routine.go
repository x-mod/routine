package routine

import (
	"context"
	"io"
	"os"
	"os/signal"
	"runtime/trace"
	"sync"

	"github.com/x-mod/errors"
)

type options struct {
	args       []interface{}
	interrupts []Interruptor
	prepare    Executor
	cleanup    Executor
	childs     []Executor
	traceOut   io.Writer
}

//Opt interface
type Opt func(*options)

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

//Prepare Opt for Main
func Prepare(exec Executor) Opt {
	return func(opts *options) {
		opts.prepare = exec
	}
}

//Execute Opt for Main
func Go(exec Executor) Opt {
	return func(opts *options) {
		if exec != nil {
			opts.childs = append(opts.childs, exec)
		}
	}
}

//Cleanup Opt for Main
func Cleanup(exec Executor) Opt {
	return func(opts *options) {
		opts.cleanup = exec
	}
}

//Trace support
func Trace(wr io.Writer) Opt {
	return func(opts *options) {
		opts.traceOut = wr
	}
}

//Routine Definition
type Routine struct {
	opts  *options
	exec  Executor
	stop  chan bool
	group sync.WaitGroup
}

//New a routine instance
func New(exec Executor, opts ...Opt) *Routine {
	mopts := &options{}
	for _, opt := range opts {
		opt(mopts)
	}
	return &Routine{
		opts: mopts,
		exec: exec,
	}
}

//Execute running the routine
func (r *Routine) Execute(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context required")
	}

	//already running
	if r.stop != nil {
		return nil
	}
	r.stop = make(chan bool)

	if err := r.prepare(ctx); err != nil {
		return err
	}
	defer r.cleanup(ctx)

	ctx, task := trace.NewTask(ctx, "main/Go")
	defer task.End()

	//argments
	if len(r.opts.args) > 0 {
		ctx = WithArgments(ctx, r.opts.args...)
	}

	// signals
	sigchan := make(chan os.Signal)
	sighandlers := make(map[os.Signal]InterruptHandler)
	for _, interruptor := range r.opts.interrupts {
		trace.Logf(ctx, "routine", "signal on %v", interruptor.Signal())
		signal.Notify(sigchan, interruptor.Signal())
		sighandlers[interruptor.Signal()] = interruptor.Interrupt()
	}

	// executor
	ch := r.mainGo(ctx, r.exec)

	// go executors
	for _, exec := range r.opts.childs {
		r.childGo(ctx, exec)
	}
	// signals outside
	for {
		select {
		case err := <-ch:
			trace.Logf(ctx, "routine", "main executor: %v", err)
			return err
		case <-r.stop:
			trace.Log(ctx, "routine", "stop invoked")
			return nil
		case <-ctx.Done():
			trace.Logf(ctx, "routine", "context done: %v", ctx.Err())
			return ctx.Err()
		case sig := <-sigchan:
			// cancel when a signal catched
			if h, ok := sighandlers[sig]; ok {
				trace.Logf(ctx, "routine", "signal catched: %v", sig)
				if h(ctx) {
					return errors.Errorf("sig %v", sig)
				}
			}
		}
	}
}

//Stop trigger the routine to stop
func (r *Routine) Stop() {
	if r.stop != nil {
		close(r.stop)
		r.stop = nil
	}
}

func (r *Routine) prepare(ctx context.Context) error {
	if r.opts.traceOut != nil {
		if err := trace.Start(r.opts.traceOut); err != nil {
			return errors.Annotate(err, "trace start")
		}
	}

	if r.opts.prepare != nil {
		ctx, task := trace.NewTask(ctx, "prepare")
		defer task.End()
		if err := r.opts.prepare.Execute(ctx); err != nil {
			trace.Logf(ctx, "routine", "prepare failed: %v", err)
			return err
		}
	}
	return nil
}

func (r *Routine) cleanup(ctx context.Context) {
	r.group.Wait()
	if r.opts.cleanup != nil {
		ctx, task := trace.NewTask(ctx, "cleanup")
		defer task.End()
		if err := r.opts.cleanup.Execute(ctx); err != nil {
			trace.Logf(ctx, "routine", "cleanup failed: %v", err)
		}
	}
	if trace.IsEnabled() {
		trace.Stop()
	}
}

func (r *Routine) mainGo(ctx context.Context, exec Executor) chan error {
	ch := make(chan error, 1)
	r.group.Add(1)
	go func() {
		defer r.group.Done()
		trace.WithRegion(ctx, "Executor", func() {
			ch <- exec.Execute(ctx)
		})
	}()
	return ch
}

func (r *Routine) childGo(ctx context.Context, exec Executor) chan error {
	ctx, task := trace.NewTask(ctx, "child/Go")
	ch := make(chan error, 1)
	r.group.Add(1)
	go func() {
		defer r.group.Done()
		defer task.End()
		trace.WithRegion(ctx, "Executor", func() {
			ch <- exec.Execute(ctx)
		})
	}()
	return ch
}
