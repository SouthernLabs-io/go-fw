package distributedlock

import (
	"context"
	"sync"
	"time"

	"github.com/southernlabs-io/go-fw/di"
	"github.com/southernlabs-io/go-fw/errors"
)

var ErrCodeLockNotAutoExtended = "LOCK_NOT_AUTO_EXTENDED"

type Factory interface {
	NewDistributedLock(resource string, ttl time.Duration) DistributedLock
}

/*
DistributedLock should be used to lock a resource across multiple processes.

Example for one time use:

	func DoSomethingWithLock(ctx context.Context) {
		dLock := distributedlock.NewPostgresDistributedLock("myResource", 10*time.Minute)
		err := dLock.Lock(ctx)
		if err != nil {
			// Handle error
			return
		}
		defer dLock.Unlock(ctx)
		doSomething()
	}

Example for long-running worker:

	func DoSomethingWithLock(ctx context.Context) {
		dLock := distributedlock.NewPostgresDistributedLock("myResource", 10*time.Minute)
		err := dLock.Lock(ctx)
		if err != nil {
			// Handle error
			return
		}
		defer dLock.Unlock(ctx)

		for {
			if time.Now().Add(5*time.Minute).After(dLock.Expiration()) {
				// Lock will expire in less than 5 minutes, extend it

				extended, err := dLock.Extend(ctx)
				if err != nil {
					// Handle error
					return
				}

				if !extended && dLock.Expiration().Before(time.Now()) {
					// Lock expired
					return
				}
			}
			doSomething()
		}
	}
*/

type DistributedLock interface {
	Resource() string
	TTL() time.Duration
	Expiration() time.Time
	ExtendedCount() int

	Lock(ctx context.Context) error
	TryLock(ctx context.Context) (bool, error)
	Unlock(ctx context.Context) error

	Extend(ctx context.Context) (bool, error)
	AutoExtend(ctx context.Context) (context.Context, error)
}

type BaseDistributedLock struct {
	mu                 sync.RWMutex
	resource           string
	id                 string
	ttl                time.Duration
	expiration         time.Time
	extendedCount      int
	autoExtenderCancel context.CancelCauseFunc
}

func (dl *BaseDistributedLock) Resource() string {
	return dl.resource
}

func (dl *BaseDistributedLock) TTL() time.Duration {
	return dl.ttl
}

func (dl *BaseDistributedLock) Expiration() time.Time {
	dl.mu.RLock()
	defer dl.mu.RUnlock()
	return dl.expiration
}

func (dl *BaseDistributedLock) ExtendedCount() int {
	dl.mu.RLock()
	defer dl.mu.RUnlock()
	return dl.extendedCount
}

// setExpiration sets the lock expiration time with proper locking
func (dl *BaseDistributedLock) setExpiration(expiration time.Time) {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	dl.expiration = expiration
}

// setLockState sets both expiration and extended count atomically
func (dl *BaseDistributedLock) setLockState(expiration time.Time, extendedCount int) {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	dl.expiration = expiration
	dl.extendedCount = extendedCount
}

// resetLockState resets the lock state (sets expiration to zero and extended count to 0)
func (dl *BaseDistributedLock) resetLockState() {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	dl.expiration = time.Time{}
	dl.extendedCount = 0
}

// setAutoExtenderCancel sets the auto extender cancel function with proper locking
func (dl *BaseDistributedLock) setAutoExtenderCancel(cancel context.CancelCauseFunc) {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	dl.autoExtenderCancel = cancel
}

// cancelAndResetAutoExtender safely cancels the auto extender and clears the reference
func (dl *BaseDistributedLock) cancelAndResetAutoExtender(cause error) {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	if dl.autoExtenderCancel != nil {
		dl.autoExtenderCancel(cause)
		dl.autoExtenderCancel = nil
	}
}

func autoExtend(ctx context.Context, dl DistributedLock, baseDL *BaseDistributedLock) (context.Context, error) {
	if ctx.Err() != nil {
		return ctx, context.Cause(ctx)
	}

	ttl := dl.TTL()
	if ttl <= 0 {
		return nil, errors.Newf(
			errors.ErrCodeBadState,
			"could not auto extend lock: %s, invalid ttl: %s",
			dl.Resource(),
			ttl,
		)
	}

	if dl.Expiration().IsZero() {
		return nil, errors.Newf(
			errors.ErrCodeBadState,
			"could not auto extend lock: %s, it is not locked!",
			dl.Resource(),
		)
	}

	ctx, cancel := context.WithCancelCause(ctx)
	baseDL.setAutoExtenderCancel(cancel)

	// Keep a minimum lead time of 100ms so scheduler/GC pauses are less likely to miss renewal.
	lead := min(max(ttl/2, 100*time.Millisecond), ttl)

	go func() {
		for {
			expiration := dl.Expiration()
			if expiration.IsZero() {
				cancel(errors.Newf(
					ErrCodeLockNotAutoExtended,
					"could not extend lock: %s, lock state was reset",
					dl.Resource(),
				))
				return
			}

			wait := time.Until(expiration.Add(-lead))
			if wait < 0 {
				wait = 0
			}

			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return
			case <-timer.C:
				extend, err := dl.Extend(ctx)
				if err != nil {
					cancel(err)
					return
				}
				if !extend {
					cancel(errors.Newf(
						ErrCodeLockNotAutoExtended,
						"could not extend lock: %s, another worker must have taken it!",
						dl.Resource(),
					))
					return
				}
			}
		}
	}()

	return ctx, nil
}

var FxExportRedis = di.FxProvideAs[Factory](NewRedisFactory, nil, nil)

var FxExportPostgresGORM = di.FxProvideAs[Factory](NewPostgresGORMFactory, nil, nil)
var FxExportPostgresBun = di.FxProvideAs[Factory](NewPostgresBunFactory, nil, nil)

var FxExportLocal = di.FxProvideAs[Factory](NewLocalFactory, nil, nil)
