routine
===

dedicated goroutine managment for `go main`, `go func`, `go routine pool`, `go crontab jobs`.

- go main
- go func
- go routine pool
- go crontab jobs

## Quick Start

In `routine` package, it use the `Executor` interface or `ExecutorFunc` instance for your implemention. 

````go

type Executor interface{
    Execute(context.Context, ...interface{})
}

type ExecutorFunc func(context.Context, ...interface{})

````

### Go Main

`routine.Main` is the basic function, when use the `routine` package. The `routine.Main` does the following things for you:

- arguments from context
- support signal interupts
- support context wait & cancel


````go
import "github.com/x-mod/routine"

func main(){
    routine.Main(routine.WithArguments(context.TODO(), "first arg", "second", false), ExecutorFunc(func(ctx context.Context, args ...interface{}){
        //out put args
        log.Println(args...)

    }), routine.DefaultCancelInterruptors...)
}

# output
# first arg second false
````

**define your own signal interruptor**

````go
// InterruptHandler definition
type InterruptHandler func(ctx context.Context, cancel context.CancelFunc) (exit bool)

// Interruptor definition
type Interruptor interface {
	Signal() os.Signal
	Interrupt() InterruptHandler
}
````


### Go Func

`routine.Go` is the wrapper for the system keyword `go`, this function should used in the `routine.Main` scope. It does the following this:

- sync.wait Add & Done
- context.Context Done check for executor go routine

````go
import "github.com/x-mod/routine"

func main(){
    routine.Main(context.TODO(), ExecutorFunc(func(ctx context.Context, args ...interface{}){

        routine.Go(routine.WithArguments(ctx, args1...), Executor1)
        routine.Go(routine.WithArguments(ctx, args2...), Executor2)

    }), routine.DefaultCancelInterruptors...)
}
````

### Go routine pool

`routine.Pool` is the go routine pool manager. you should use it in `routine.Main` scope either, for the `routine.Main` controls the routines exiting events. And the `routine.Pool` does the following things for you:

- go routines management, like auto create new routine & release idle routine
- support fixed Executor & dynamic Executor
- async invoke functions

**dynamic executor example**:

````go
import "github.com/x-mod/routine"

func main(){

    routine.Main(context.Backgroud(), ExecutorFunc(func(ctx context.Context, args ...interface{}){
        //dynamic executors pool
        pool := routine.NewPool(routine.RunningSize(4), routine.WatingSize(8))
       
        //open
        if err := pool.Open(ctx); err != nil {
            //TODO
            return
        }
        //close
        defer pool.Close()

        //async invoke multiple dynamic executors
        pool.Go(routine.WithArguments(ctx, args1...), executor1)
        pool.Go(routine.WithArguments(ctx, args2...), executor2)

    }), routine.DefaultCancelInterruptors...)
}
````

**fixed executor example**:

````go
import "github.com/x-mod/routine"

func main(){

    routine.Main(context.Backgroud(), ExecutorFunc(func(ctx context.Context, args ...interface{}){
       
        //fixed executor pool
        fixedPool := routine.NewPool(routine.RunningSize(4), 
            routine.WatingSize(8),
            routine.FixedExecutor(executor3))
        
        //open
        if err := fixedPool.Open(ctx); err != nil {
            //TODO
            return
        }
        //close
        defer fixedPool.Close()

        //async invoke fixed executor
        fixedPool.Execute(ctx, args1...)
        fixedPool.Execute(ctx, args2...)

    }), routine.DefaultCancelInterruptors...)
}
````

### Go crontab jobs

`routine.Crontab` is similar interface like linux system's crontab jobs. You can 

````go
import "github.com/x-mod/routine"

func main(){
    crontab := routine.NewCrontab(routine.RunningSize(4))
    defer crontab.Close()

    routine.Main(context.Backgroud(), ExecutorFunc(func(ctx context.Context, args ...interface{}){
        
        //open crontab
        if err := crontab.Open(ctx); err != nil {
            //TODO
            return
        }

        // crontab format schedule
        crontab.JOB("* * * * *", executor1).Go(ctx, args1...)
        crontab.JOB("* * * * *", executor2).Go(ctx, args2...)

        // every interval
        crontab.EVERY(time.Second, executor3).Go(ctx, args3 ...)
        crontab.EVERY(time.Minute, executor4).Go(ctx, args4 ...)

        // now, run executor at once
        crontab.NOW(executor5).Go(ctx, args5...)

    }), routine.DefaultCancelInterruptors...)
}
````

