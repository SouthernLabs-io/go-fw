package distributedlock

import (
	"context"
	"crypto/sha1"
	"fmt"
	"math/rand"
	"os"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
	fwsync "github.com/southernlabs-io/go-fw/sync"
)

type LocalFactory struct {
}

func NewLocalFactory() *LocalFactory {
	return &LocalFactory{}
}

func (f *LocalFactory) NewDistributedLock(resource string, ttl time.Duration) DistributedLock {
	return NewDistributedLocalLock(resource, ttl)
}

type LocalLock struct {
	BaseDistributedLock

	path   string
	fd     int
	locked bool
	mu     *sync.Mutex
}

var _ DistributedLock = &LocalLock{}

func NewDistributedLocalLock(resource string, ttl time.Duration) *LocalLock {
	dir := path.Join(os.TempDir(), "_go-fw", "local_lock")
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		panic(errors.NewUnknownf("failed to create lock directory: %s, error: %w", dir, err))
	}

	filePath := path.Join(dir, fmt.Sprintf("%x", sha1.Sum([]byte(resource))))

	return &LocalLock{
		BaseDistributedLock: BaseDistributedLock{
			resource: resource,
			id:       uuid.NewString(),
			ttl:      ttl,
		},
		path: filePath,
		fd:   -1,
		mu:   &sync.Mutex{},
	}
}

func (l *LocalLock) Lock(ctx context.Context) error {
	var locked bool
	for {
		var err error
		locked, err = l.TryLock(ctx)
		if err != nil {
			return err
		}
		if locked {
			return nil
		}

		jitter := time.Duration(0)
		if l.ttl > 10 {
			jitter = time.Duration(rand.Int63n(int64(l.ttl) / 10))
		}
		sleepDuration := max(l.ttl/10+jitter, 10*time.Millisecond)
		err = fwsync.Sleep(ctx, sleepDuration)
		if err != nil {
			return err
		}
	}
}

func (l *LocalLock) doLock(_ context.Context) {
	l.setLockState(time.Now().Add(l.ttl), 0)
	l.locked = true

	go func() {
		for {
			l.mu.Lock()
			if !l.locked {
				l.mu.Unlock()
				return
			}
			exp := l.Expiration()
			remaining := time.Until(exp)
			if remaining <= 0 {
				_ = l.unlockLocked(context.Background())
				l.mu.Unlock()
				return
			}
			l.mu.Unlock()

			sleepDuration := remaining / 2
			if sleepDuration < 10*time.Millisecond {
				sleepDuration = remaining
			}
			if sleepDuration <= 0 {
				sleepDuration = 10 * time.Millisecond
			}
			time.Sleep(sleepDuration)
		}
	}()
}

func (l *LocalLock) TryLock(ctx context.Context) (bool, error) {
	logger := log.GetLoggerFromCtx(ctx)

	if !l.mu.TryLock() {
		logger.Debugf("Lock not acquired: %s, lockID: %s", l.resource, l.id)
		return false, nil
	}
	defer l.mu.Unlock()

	if l.locked {
		logger.Debugf("Lock not acquired: %s, lockID: %s", l.resource, l.id)
		return false, nil
	}

	fd, err := syscall.Open(l.path, syscall.O_CREAT|syscall.O_RDWR, 0666)
	if err != nil {
		return false, errors.NewUnknownf("failed to open lock file: %s, error: %w", l.path, err)
	}

	err = syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = syscall.Close(fd)
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			logger.Debugf("Lock not acquired: %s, lockID: %s", l.resource, l.id)
			return false, nil
		}
		return false, errors.NewUnknownf("failed to lock: %s file: %s, error: %w", l.resource, l.path, err)
	}

	l.fd = fd
	l.doLock(ctx)
	logger.Debugf("Lock acquired: %s, lockID: %s, expiration: %s", l.resource, l.id, l.Expiration())
	return true, nil
}

func (l *LocalLock) Unlock(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.unlockLocked(ctx)
}

func (l *LocalLock) unlockLocked(ctx context.Context) error {
	if !l.locked {
		return nil
	}

	var flockErr, closeErr error
	if l.fd >= 0 {
		flockErr = syscall.Flock(l.fd, syscall.LOCK_UN)
		closeErr = syscall.Close(l.fd)
		l.fd = -1
	}

	l.cancelAndResetAutoExtender(context.Canceled)
	l.resetLockState()
	l.locked = false

	if flockErr != nil {
		return errors.NewUnknownf("failed to unlock file: %s, error: %w", l.path, flockErr)
	}
	if closeErr != nil {
		return errors.NewUnknownf("failed to close lock file: %s, error: %w", l.path, closeErr)
	}

	if ctx != nil && ctx.Err() == nil {
		log.GetLoggerFromCtx(ctx).Debugf("Lock unlocked: %s, lockID: %s, file: %s", l.resource, l.id, l.path)
	}
	return nil
}

func (l *LocalLock) Extend(ctx context.Context) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.locked {
		return false, nil
	}

	if time.Now().After(l.Expiration()) {
		l.resetLockState()
		return false, nil
	}

	expiration := time.Now().Add(l.ttl)
	count := l.ExtendedCount() + 1
	l.setLockState(expiration, count)
	log.GetLoggerFromCtx(ctx).Tracef(
		"Lock extended: %s, lockID: %s, expiration: %s, extendedCount: %d",
		l.resource,
		l.id,
		expiration,
		count,
	)
	return true, nil
}

func (l *LocalLock) AutoExtend(ctx context.Context) (context.Context, error) {
	return autoExtend(ctx, l, &l.BaseDistributedLock)
}
