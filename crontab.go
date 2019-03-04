package routine

import (
	"context"
	"time"

	"github.com/gorhill/cronexpr"
)

//Crontab def
type Crontab struct {
	pool *Pool
	jobs []*Job
}

//NewCrontab create a new crontab instance
func NewCrontab(opts ...Option) *Crontab {
	return &Crontab{
		pool: NewPool(opts...),
		jobs: []*Job{},
	}
}

//Open start crontab working
func (crontab *Crontab) Open(ctx context.Context) error {
	return crontab.pool.Open(ctx)
}

//Close all crontab jobs
func (crontab *Crontab) Close() {
	for _, job := range crontab.jobs {
		job.Close()
	}
	crontab.pool.Close()
}

//JOB a new job
// schedule format just same as crontab format
// * 	* 	 * 	  * 	*
// min  hour day  month weekday
func (crontab *Crontab) JOB(schedule string, exec Executor) *Job {
	exp, err := cronexpr.Parse(schedule)
	if err != nil {
		panic(err)
	}
	job := &Job{
		crontab:  crontab,
		schedule: schedule,
		expr:     exp,
		exec:     exec,
		err:      err,
		stop:     make(chan bool),
	}
	crontab.jobs = append(crontab.jobs, job)
	return job
}

//EVERY duration job
func (crontab *Crontab) EVERY(d time.Duration, exec Executor) *Job {
	job := &Job{
		crontab:  crontab,
		duration: d,
		exec:     exec,
		stop:     make(chan bool),
	}
	crontab.jobs = append(crontab.jobs, job)
	return job
}

//NOW job
func (crontab *Crontab) NOW(exec Executor) *Job {
	job := &Job{
		crontab: crontab,
		atOnce:  true,
		exec:    exec,
		stop:    make(chan bool),
	}
	return job
}

//Job def
type Job struct {
	crontab  *Crontab
	schedule string
	expr     *cronexpr.Expression
	duration time.Duration
	atOnce   bool
	exec     Executor
	err      error
	stop     chan bool
}

func (job *Job) next() time.Time {
	if job.expr != nil {
		return job.expr.Next(time.Now())
	}
	if job.duration > time.Duration(0) {
		return time.Now().Add(job.duration)
	}
	return time.Time{}
}

//Go start a new crontab job by executor func
func (job *Job) Go(ctx context.Context, args ...interface{}) error {
	if job.err != nil {
		return job.err
	}
	if job.atOnce {
		job.crontab.pool.Go(WithArgments(ctx, args...), job.exec)
		return nil
	}

	Go(WithArgments(ctx, args...), ExecutorFunc(func(c context.Context, args ...interface{}) {
		for {
			tm := job.next()
			if !tm.IsZero() {
				select {
				case <-job.stop:
					return
				case <-ctx.Done():
					return
				case <-time.After(tm.Sub(time.Now())):
					job.crontab.pool.Go(c, job.exec)
				}
			}
		}
	}))
	return nil
}

//Close job
func (job *Job) Close() {
	close(job.stop)
}
