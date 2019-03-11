package routine

import (
	"context"
	"runtime"
	"sync"
	"time"
)

//Pool goroutines pool
type Pool struct {
	mu          sync.Mutex
	cf          *config
	running     int
	rootCtx     context.Context
	waitRunners chan *runner
}

type runner struct {
	ctx context.Context
	exe Executor
}

//NewPool new goroutine pool instance
func NewPool(opts ...Option) *Pool {
	cfg := &config{
		runningSize:   2,
		waitingSize:   4,
		clearDuration: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &Pool{
		cf:          cfg,
		waitRunners: make(chan *runner, cfg.waitingSize),
	}
}

func (p *Pool) clear() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return (p.running > p.cf.runningSize)
}

func (p *Pool) overload() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cf.maxCurrency > 0 {
		return (p.running >= p.cf.maxCurrency)
	}
	return false
}

func (p *Pool) increment() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.running = p.running + 1
}

func (p *Pool) decrement() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.running = p.running - 1
}

func (p *Pool) execute(ctx context.Context, args ...interface{}) {
	p.increment()
	defer p.decrement()
	for {
		select {
		case <-time.After(p.cf.clearDuration):
			if p.clear() {
				return
			}
		case <-ctx.Done():
			return
		case runner, ok := <-p.waitRunners:
			if !ok {
				return
			}
			if runner.exe != nil {
				argments := ArgumentsFrom(runner.ctx)
				runner.exe.Execute(runner.ctx, argments...)
			}
		}
	}
}

//Open start pool goroutines
func (p *Pool) Open(ctx context.Context) error {
	p.rootCtx = ctx
	for i := 0; i < p.cf.runningSize; i++ {
		Go(p.rootCtx, ExecutorFunc(p.execute))
	}
	return nil
}

//Close stop pool goroutines
func (p *Pool) Close() {
	close(p.waitRunners)
}

//Go get an idle goroutine to do the executor immediately
func (p *Pool) Go(ctx context.Context, exec Executor) {
	if exec != nil {
		select {
		case p.waitRunners <- &runner{ctx, exec}:
			break
		default:
			if !p.overload() {
				//waitRunners is full, start a new goroutine
				Go(p.rootCtx, ExecutorFunc(p.execute))
				runtime.Gosched()
			}
			//blocked when overloaded
			p.waitRunners <- &runner{ctx, exec}
		}
	}
}

//Execute only valid when FixedExecutor(exec) Set
func (p *Pool) Execute(ctx context.Context, args ...interface{}) {
	if p.cf.executor == nil {
		panic("unset fixed executor")
	}
	p.Go(WithArgments(ctx, args...), p.cf.executor)
}
