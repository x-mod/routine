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
	args         []interface{}
	interrupts   []Interruptor
	prepareBlock bool
	prepareExec  Executor
	cleanupExec  Executor
	executors    []Executor
	traceOut     io.Writer
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
		opts.prepareExec = exec
	}
}

//Execute Opt for Main
func Go(exec Executor) Opt {
	return func(opts *options) {
		if exec != nil {
			opts.executors = append(opts.executors, exec)
		}
	}
}

//Cleanup Opt for Main
func Cleanup(exec Executor) Opt {
	return func(opts *options) {
		opts.cleanupExec = exec
	}
}

//Trace support
func Trace(wr io.Writer) Opt {
	return func(opts *options) {
		opts.traceOut = wr
	}
}

type Routine struct {
	opts  *options
	exec  Executor
	stop  chan bool
	group sync.WaitGroup
}

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

func (r *Routine) Execute(ctx context.Context) error {
	defer r.close(ctx)
	if r.opts.traceOut != nil {
		if err := trace.Start(r.opts.traceOut); err != nil {
			return errors.Annotate(err, "trace start")
		}
	}

	ctx, task := trace.NewTask(ctx, "Main")
	defer task.End()
	//already running
	if r.stop != nil {
		return nil
	}
	if ctx == nil {
		return errors.New("context required")
	}
	rctx := ctx
	r.stop = make(chan bool)

	//argments
	if len(r.opts.args) > 0 {
		rctx = WithArgments(rctx, r.opts.args...)
	}
	//prepare
	if r.opts.prepareExec != nil {
		defer trace.StartRegion(ctx, "Prepare").End()
		if err := r.opts.prepareExec.Execute(ctx); err != nil {
			trace.Logf(ctx, "Main", "prepare execute failed: %v", err)
			return err
		}
	}
	//cleanup defer
	defer func() {
		if r.opts.cleanupExec != nil {
			trace.WithRegion(ctx, "Cleanup", func() {
				if err := r.opts.cleanupExec.Execute(rctx); err != nil {
					trace.Logf(ctx, "Main", "cleanup execute failed: %v", err)
				}
			})
		}
	}()

	// signals
	sigchan := make(chan os.Signal)
	sighandlers := make(map[os.Signal]InterruptHandler)
	for _, interruptor := range r.opts.interrupts {
		trace.Logf(ctx, "Main", "signal on %v", interruptor.Signal())
		signal.Notify(sigchan, interruptor.Signal())
		sighandlers[interruptor.Signal()] = interruptor.Interrupt()
	}

	// executor
	ch := r.mainGo(rctx, r.exec)

	// go executors
	for _, exec := range r.opts.executors {
		r.childGo(rctx, exec)
	}
	// signals outside
	for {
		select {
		case err := <-ch:
			trace.Logf(ctx, "Main", "exiting for excutor stopped: %v", err)
			return err
		case <-r.stop:
			trace.Log(rctx, "Main", "stop invoked")
			return nil
		case <-rctx.Done():
			trace.Logf(ctx, "Main", "exiting for context done: %v", rctx.Err())
			return rctx.Err()
		case sig := <-sigchan:
			// cancel when a signal catched
			if h, ok := sighandlers[sig]; ok {
				if h(rctx) {
					trace.Logf(ctx, "Main", "exiting for sig catched: %v", sig)
					return errors.Errorf("sig %v", sig)
				}
			}
		}
	}
}

func (r *Routine) close(ctx context.Context) {
	if r.stop != nil {
		close(r.stop)
		r.stop = nil
	}
	r.group.Wait()
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
	ctx, task := trace.NewTask(ctx, "Go")
	ch := make(chan error, 1)
	r.group.Add(1)
	go func() {
		defer task.End()
		defer r.group.Done()
		trace.WithRegion(ctx, "Executor", func() {
			ch <- exec.Execute(ctx)
		})
	}()
	return ch
}
