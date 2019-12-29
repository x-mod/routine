package routine

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/x-mod/errors"
)

type options struct {
	args        []interface{}
	interrupts  []Interruptor
	prepareExec Executor
	cleanupExec Executor
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

//Cleanup Opt for Main
func Cleanup(exec Executor) Opt {
	return func(opts *options) {
		opts.cleanupExec = exec
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
		if err := r.opts.prepareExec.Execute(rctx); err != nil {
			return errors.Annotate(err, "routine prepare")
		}
	}
	//clearup defer
	defer func() {
		if r.opts.cleanupExec != nil {
			if err := r.opts.cleanupExec.Execute(rctx); err != nil {
				log.Println("routine clearup failed:", err)
			}
		}
	}()

	// signals
	sigchan := make(chan os.Signal)
	sighandlers := make(map[os.Signal]InterruptHandler)
	for _, interruptor := range r.opts.interrupts {
		signal.Notify(sigchan, interruptor.Signal())
		sighandlers[interruptor.Signal()] = interruptor.Interrupt()
	}

	// executor
	ch := r.Go(rctx, r.exec)

	// signals outside
	for {
		select {
		case err := <-ch:
			return err
		case <-r.stop:
			return nil
		case <-rctx.Done():
			return rctx.Err()
		case sig := <-sigchan:
			// cancel when a signal catched
			if h, ok := sighandlers[sig]; ok {
				if h(rctx) {
					return errors.Errorf("sig %v", sig)
				}
			}
		}
	}
}

func (r *Routine) Stop() {
	if r.stop != nil {
		close(r.stop)
		r.stop = nil
	}
	r.group.Wait()
}

func (r *Routine) Go(ctx context.Context, exec Executor) chan error {
	ch := make(chan error)
	r.group.Add(1)
	go func() {
		defer r.group.Done()
		ch <- exec.Execute(ctx)
	}()
	return ch
}
