package routine

import "time"

type config struct {
	runningSize   int
	waitingSize   int
	maxCurrency   int
	clearDuration time.Duration
	executor      Executor
}

//Option def
type Option func(*config)

//RunningSize Pool Option for running goroutine size
func RunningSize(sz int) Option {
	return func(cfg *config) {
		cfg.runningSize = sz
	}
}

//WaitingSize Pool Option for executor request buffer size
func WaitingSize(sz int) Option {
	return func(cfg *config) {
		cfg.waitingSize = sz
	}
}

//MaxCurrency Pool Option for max current running goroutine size
func MaxCurrency(sz int) Option {
	return func(cfg *config) {
		cfg.maxCurrency = sz
	}
}

//ClearDuration Pool Option for duration clear idling goroutines
func ClearDuration(d time.Duration) Option {
	return func(cfg *config) {
		cfg.clearDuration = d
	}
}

//FixedExecutor Pool Option for fixed executor
func FixedExecutor(exec Executor) Option {
	return func(cfg *config) {
		if exec != nil {
			cfg.executor = exec
		}
	}
}
