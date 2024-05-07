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

type WaitGroup = sync.WaitGroup

type Once = sync.Once

type Mutex = sync.Mutex

type RWMutex = sync.RWMutex

type Cond = sync.Cond

type Pool = sync.Pool

// # endregion
