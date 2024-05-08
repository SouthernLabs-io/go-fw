package sync

import (
	"context"
	"sync"
	"time"
)

// Sleep sleeps for the specified duration or until the context is done.
// If it was the context that was done first, it returns the error that caused it using context.Cause.
func Sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

// # region copy of sync go std package

// Types
type (
	WaitGroup = sync.WaitGroup
	Once      = sync.Once
	Mutex     = sync.Mutex
	RWMutex   = sync.RWMutex
	Cond      = sync.Cond
	Pool      = sync.Pool
)

// Functions
var (
	NewCond  = sync.NewCond
	OnceFunc = sync.OnceFunc
)

func OnceValue[T any](f func() T) func() T {
	return sync.OnceValue(f)
}

func OnceValues[T1, T2 any](f func() (T1, T2)) func() (T1, T2) {
	return sync.OnceValues(f)
}

// # endregion
