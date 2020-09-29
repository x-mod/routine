package routine

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime/trace"
	"sync"

	"github.com/x-mod/errors"
	"github.com/x-mod/sigtrap"
)

type options struct {
	args     []interface{}
	prepare  Executor
	cleanup  Executor
	childs   []Executor
	traceOut io.Writer
	sigtraps []*SigTrap
	csignals []os.Signal
}

//Opt interface
type Opt func(*options)

//Arguments Opt for Main
func Arguments(args ...interface{}) Opt {
	return func(opts *options) {
		opts.args = args
	}
}

type SigHandler func()
type SigTrap struct {
	sig     os.Signal
	handler SigHandler
}

//Signal Opt for Main
func Signal(sig os.Signal, handler SigHandler) Opt {
	return func(opts *options) {
		if handler != nil {
			opts.sigtraps = append(opts.sigtraps, &SigTrap{sig: sig, handler: handler})
		}
	}
}

//CancelSignals Opt for Main
func CancelSignals(sigs ...os.Signal) Opt {
	return func(opts *options) {
		opts.csignals = append(opts.csignals, sigs...)
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
	caps  *sigtrap.Capture
}

//New a routine instance
func New(exec Executor, opts ...Opt) *Routine {
	mopts := &options{
		args:     []interface{}{},
		sigtraps: []*SigTrap{},
		csignals: []os.Signal{},
	}
	for _, opt := range opts {
		opt(mopts)
	}
	routine := &Routine{
		opts: mopts,
		exec: exec,
	}
	return routine
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

	//signals
	if len(r.opts.csignals) > 0 {
		cctx, cancel := context.WithCancel(ctx)
		handler := SigHandler(cancel)
		for _, sig := range r.opts.csignals {
			r.opts.sigtraps = append(r.opts.sigtraps, &SigTrap{sig: sig, handler: handler})
		}
		ctx = cctx
	}
	if len(r.opts.sigtraps) > 0 {
		sopts := make([]sigtrap.CaptureOpt, 0, len(r.opts.sigtraps))
		for _, trap := range r.opts.sigtraps {
			sopts = append(sopts, sigtrap.Trap(trap.sig, sigtrap.Handler(trap.handler)))
		}
		r.caps = sigtrap.New(sopts...)
		r.childGo("sigtrap", ctx, ExecutorFunc(r.caps.Serve))
	}

	// executor
	ch := r.mainGo(ctx, r.exec)

	// go executors
	for _, exec := range r.opts.childs {
		r.childGo("child", ctx, exec)
	}
	// signals outside
	select {
	case err := <-ch:
		trace.Logf(ctx, "routine", "main executor: %v", err)
		if r.caps != nil {
			r.caps.Close()
		}
		return err
	case <-r.stop:
		if r.caps != nil {
			r.caps.Close()
		}
		trace.Log(ctx, "routine", "stop invoked")
		return nil
	case <-ctx.Done():
		trace.Logf(ctx, "routine", "context done: %v", ctx.Err())
		return ctx.Err()
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

func (r *Routine) childGo(name string, ctx context.Context, exec Executor) chan error {
	ctx, task := trace.NewTask(ctx, fmt.Sprintf("%s/Go", name))
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
