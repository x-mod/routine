package routine

import (
	"context"
	"sync"

	"github.com/x-mod/errors"
)

const defaultNumOfRoutines = 4
const defaultMaxRequestBufferSize = 8

//Pool struct
type Pool struct {
	num   int
	max   int
	ctx   context.Context
	reqs  chan *request
	open  bool
	close chan bool
	mu    sync.Mutex
}

type request struct {
	ctx  context.Context
	exec Executor
	ch   chan error
}

//PoolOpt opt for pool
type PoolOpt func(*Pool)

//NumOfRoutines opt
func NumOfRoutines(n int) PoolOpt {
	return func(p *Pool) {
		if n > 0 {
			p.num = n
		}
	}
}

//MaxOfRequestBufferSize opt
func MaxOfRequestBufferSize(n int) PoolOpt {
	return func(p *Pool) {
		if n > 0 {
			p.max = n
		}
	}
}

//NewPool new routine pool
func NewPool(opts ...PoolOpt) *Pool {
	p := &Pool{
		num:   defaultNumOfRoutines,
		max:   defaultMaxRequestBufferSize,
		ctx:   context.TODO(),
		open:  false,
		close: make(chan bool),
	}
	for _, o := range opts {
		o(p)
	}
	p.reqs = make(chan *request, p.max)
	return p
}

//Open starting the pool routines
func (p *Pool) Open(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.open {
		if ctx != nil {
			p.ctx = ctx
		}
		WithWait(p.ctx)
		WaitAdd(p.ctx, p.num)
		for i := 0; i < p.num; i++ {
			go p.exec(p.ctx)
		}
		p.open = true
	}
}

func (p *Pool) exec(ctx context.Context) error {
	defer WaitDone(p.ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.close: //clear of requests
			for r := range p.reqs {
				r.ch <- errors.New("pool closed")
			}
			return nil
		case r := <-p.reqs:
			// DO NOT, the following implemention means pool is fake
			// r.ch <- <-Go(r.ctx, r.exec)
			// NOTICE: POOL only check context before Executor executing
			select {
			case <-r.ctx.Done():
				r.ch <- r.ctx.Err()
			default:
				r.ch <- r.exec.Execute(r.ctx)
			}
		}
	}
	return nil
}

//Go impl routine interface
func (p *Pool) Go(ctx context.Context, exec Executor) chan error {
	//open pool if not
	p.Open(nil)
	//req
	ch := make(chan error, 1)
	if exec == nil {
		ch <- errors.New("Executor required")
		return ch
	}
	select {
	case <-p.ctx.Done():
		ch <- p.ctx.Err()
	default:
		req := &request{
			ctx:  context.TODO(),
			exec: exec,
			ch:   ch,
		}
		if req.ctx != nil {
			req.ctx = ctx
		}
		p.reqs <- req
	}
	return ch
}

//Close running pool
func (p *Pool) Close() {
	close(p.close)
	Wait(p.ctx)
}
