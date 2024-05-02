package distributedlock

import (
	"context"
	"time"

	"github.com/southernlabs-io/go-fw/errors"
)

var ErrCodeLockNotAutoExtended = "LOCK_NOT_AUTO_EXTENDED"

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
	return dl.expiration
}

func (dl *BaseDistributedLock) ExtendedCount() int {
	return dl.extendedCount
}

func autoExtend(ctx context.Context, dl DistributedLock, baseDL *BaseDistributedLock) (context.Context, error) {
	if ctx.Err() != nil {
		return ctx, context.Cause(ctx)
	}

	if dl.Expiration().IsZero() {
		return nil, errors.Newf(
			errors.ErrCodeBadState,
			"could not auto extend lock: %s, it is not locked!",
			dl.Resource(),
		)
	}

	ctx, cancel := context.WithCancelCause(ctx)
	baseDL.autoExtenderCancel = cancel
	ttl := dl.TTL()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(ttl / 2):
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
