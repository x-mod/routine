package routine

import (
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/x-mod/errors"
)

//GuaranteeExecutor struct, make sure of none error return
type GuaranteeExecutor struct {
	exec Executor
}

//Guarantee insure exec NEVER PANIC
func Guarantee(exec Executor) *GuaranteeExecutor {
	return &GuaranteeExecutor{exec}
}

//Execute implement Executor interface
func (g *GuaranteeExecutor) Execute(ctx context.Context) error {
	do := func(ctx context.Context) (err error) {
		defer func() {
			if rc := recover(); rc != nil {
				switch rv := rc.(type) {
				case error:
					err = rv.(error)
					return
				default:
					err = errors.Errorf("%v", rv)
					return
				}
			}
		}()
		err = g.exec.Execute(ctx)
		return
	}
	if err := do(ctx); err != nil {
		log.Println("execute failed:", err)
	}
	return nil
}

//RetryExecutor struct
type RetryExecutor struct {
	retryTimes int
	exec       Executor
}

type _retry struct{}

//FromRetry current retied times
func FromRetry(ctx context.Context) int {
	if ctx != nil {
		retried := ctx.Value(_retry{})
		if retried != nil {
			return retried.(int)
		}
	}
	return 0
}

//Retry new
func Retry(retry int, exec Executor) *RetryExecutor {
	return &RetryExecutor{
		retryTimes: retry,
		exec:       exec,
	}
}

//Execute implement Executor interface
func (retry *RetryExecutor) Execute(ctx context.Context) error {
	var err error
	if retry.retryTimes == 0 {
		retry.retryTimes = 1
	}
	for i := 0; i < retry.retryTimes; i++ {
		if err = retry.exec.Execute(context.WithValue(ctx, _retry{}, i+1)); err != nil {
			continue
		}
		return nil
	}
	return err
}

//RepeatExecutor struct
type RepeatExecutor struct {
	repeatTimes    int
	repeatInterval time.Duration
	exec           Executor
}

type _repeat struct{}

//FromRepeat current repeated times
func FromRepeat(ctx context.Context) int {
	if ctx != nil {
		repeated := ctx.Value(_repeat{})
		if repeated != nil {
			return repeated.(int)
		}
	}
	return 0
}

//Repeat new
func Repeat(repeat int, interval time.Duration, exec Executor) *RepeatExecutor {
	return &RepeatExecutor{
		repeatTimes:    repeat,
		repeatInterval: interval,
		exec:           exec,
	}
}

//Execute implement Executor
func (r *RepeatExecutor) Execute(ctx context.Context) error {
	fn := func(repeat int) error {
		if err := r.exec.Execute(context.WithValue(ctx, _repeat{}, repeat)); err != nil {
			return err
		}
		if r.repeatInterval > 0 {
			<-time.After(r.repeatInterval)
		}
		return nil
	}
	if r.repeatTimes > 0 {
		for i := 0; i < r.repeatTimes; i++ {
			if err := fn(i + 1); err != nil {
				return errors.Annotatef(err, "repeat %d failed:", i)
			}
		}
		return nil
	}

	for i := 0; ; i++ {
		if err := fn(i + 1); err != nil {
			return errors.Annotatef(err, "repeat %d failed:", i)
		}
	}
}

//CrontabExecutor struct
type CrontabExecutor struct {
	plan    string
	workday bool
	weekend bool
	mutes   []*CrontabMute
	exec    Executor
}

type CrontabMute struct {
	begin time.Time
	end   time.Time
}

type _crontab struct{}

//FromCrontab current crontab time
func FromCrontab(ctx context.Context) time.Time {
	if ctx != nil {
		crontab := ctx.Value(_crontab{})
		if crontab != nil {
			return crontab.(time.Time)
		}
	}
	return time.Time{}
}

//Crontab new
func Crontab(plan string, exec Executor) *CrontabExecutor {
	return &CrontabExecutor{
		plan: plan,
		exec: exec,
	}
}

func (c *CrontabExecutor) Weekend(flag bool) {
	c.weekend = flag
}
func (c *CrontabExecutor) Workday(flag bool) {
	c.workday = flag
}
func (c *CrontabExecutor) Everyday(flag bool) {
	if flag {
		c.weekend = false
		c.workday = false
	}
}
func (c *CrontabExecutor) Mute(begin time.Time, end time.Time) {
	m := &CrontabMute{begin: begin, end: end}
	c.mutes = append(c.mutes, m)
}

func (c *CrontabExecutor) IsTimeMuted(tm time.Time) bool {
	for _, m := range c.mutes {
		if m.begin.Before(tm) && m.end.After(tm) {
			return true
		}
	}
	switch tm.Weekday() {
	case time.Sunday, time.Saturday:
		if c.workday {
			return true
		}
	case time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday:
		if c.weekend {
			return true
		}
	}
	return false
}

//Execute implement Executor
func (c *CrontabExecutor) Execute(ctx context.Context) error {
	exp, err := cronexpr.Parse(c.plan)
	if err != nil {
		return err
	}
	next := exp.Next(time.Now())
	if next.IsZero() {
		return ErrNonePlan
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(next.Sub(time.Now())):
			if c.IsTimeMuted(next) == false {
				if err := c.exec.Execute(context.WithValue(ctx, _crontab{}, next)); err != nil {
					return err
				}
			}
			next = exp.Next(time.Now())
			if next.IsZero() {
				return ErrNonePlan
			}
		}
	}
}

//CommandExecutor struct
type CommandExecutor struct {
	command string
	args    []string
	envs    []string
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
}
type CommandOpt func(*CommandExecutor)

func ARG(arg string) CommandOpt {
	return func(c *CommandExecutor) {
		c.args = append(c.args, arg)
	}
}

func ENV(env string) CommandOpt {
	return func(c *CommandExecutor) {
		c.envs = append(c.envs, env)
	}
}

func Stdin(rd io.Reader) CommandOpt {
	return func(c *CommandExecutor) {
		c.stdin = rd
	}
}

func Stdout(wr io.Writer) CommandOpt {
	return func(c *CommandExecutor) {
		c.stdout = wr
	}
}

func Stderr(wr io.Writer) CommandOpt {
	return func(c *CommandExecutor) {
		c.stderr = wr
	}
}

//Command new
func Command(cmd string, opts ...CommandOpt) *CommandExecutor {
	ce := &CommandExecutor{
		command: cmd,
		args:    []string{},
		envs:    []string{},
		stdin:   os.Stdin,
		stdout:  os.Stdout,
		stderr:  os.Stderr,
	}
	for _, opt := range opts {
		opt(ce)
	}
	return ce
}

//Execute implement Executor
func (cmd *CommandExecutor) Execute(ctx context.Context) error {
	c := exec.CommandContext(ctx, cmd.command, cmd.args...)
	c.Env = append(c.Env, cmd.envs...)
	c.Stdin = cmd.stdin
	c.Stdout = cmd.stdout
	c.Stderr = cmd.stderr
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

//TimeoutExecutor struct
type TimeoutExecutor struct {
	timeout time.Duration
	exec    Executor
}

//Timeout new
func Timeout(d time.Duration, exec Executor) *TimeoutExecutor {
	return &TimeoutExecutor{
		timeout: d,
		exec:    exec,
	}
}

func (tm *TimeoutExecutor) Execute(ctx context.Context) error {
	tmCtx, cancel := context.WithTimeout(ctx, tm.timeout)
	defer cancel()
	return New(tm.exec).Execute(tmCtx)
}

//DeadlineExecutor struct
type DeadlineExecutor struct {
	deadline time.Time
	exec     Executor
}

//Deadline new
func Deadline(d time.Time, exec Executor) *DeadlineExecutor {
	return &DeadlineExecutor{
		deadline: d,
		exec:     exec,
	}
}

//Execute implement Executor
func (tm *DeadlineExecutor) Execute(ctx context.Context) error {
	tmCtx, cancel := context.WithDeadline(ctx, tm.deadline)
	defer cancel()
	return New(tm.exec).Execute(tmCtx)
}

//ConcurrentExecutor struct
type ConcurrentExecutor struct {
	concurrent int
	exec       Executor
	wg         sync.WaitGroup
}

//Concurrent new
func Concurrent(c int, exec Executor) *ConcurrentExecutor {
	return &ConcurrentExecutor{
		concurrent: c,
		exec:       exec,
	}
}

//Execute implement Executor
func (ce *ConcurrentExecutor) Execute(ctx context.Context) error {
	for i := 0; i < ce.concurrent; i++ {
		ce.wg.Add(1)
		go func(i int) {
			defer ce.wg.Done()
			if err := ce.exec.Execute(ctx); err != nil {
				log.Println("concurrent ", i, " failed:", err)
			}
		}(i)
	}
	ce.wg.Wait()
	return nil
}

//ParallelExecutor
type ParallelExecutor struct {
	execs []Executor
	wg    sync.WaitGroup
}

//Parallel new
func Parallel(execs ...Executor) *ParallelExecutor {
	return &ParallelExecutor{
		execs: execs,
	}
}

//Execute implement Executor
func (pe *ParallelExecutor) Execute(ctx context.Context) error {
	for i, exec := range pe.execs {
		pe.wg.Add(1)
		go func(i int, exec Executor) {
			defer pe.wg.Done()
			if err := exec.Execute(ctx); err != nil {
				log.Println("parallel ", i, " failed:", err)
			}
		}(i, exec)
	}
	pe.wg.Wait()
	return nil
}

//AppendExecutor
type AppendExecutor struct {
	execs []Executor
}

//Append new
func Append(execs ...Executor) *AppendExecutor {
	executor := &AppendExecutor{
		execs: []Executor{},
	}
	executor.execs = append(executor.execs, execs...)
	return executor
}

func (ae *AppendExecutor) Append(execs ...Executor) {
	ae.execs = append(ae.execs, execs...)
}

func (ae *AppendExecutor) Execute(ctx context.Context) error {
	for i, exec := range ae.execs {
		if err := exec.Execute(ctx); err != nil {
			return errors.Annotatef(err, "no(%d)", i)
		}
	}
	return nil
}
