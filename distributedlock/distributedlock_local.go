package distributedlock

import (
	"context"
	"crypto/sha1"
	"fmt"
	"os"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
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

func NewDistributedLocalLock(resource string, ttl time.Duration) *LocalLock {
	dir := path.Join(os.TempDir(), "_go-fw", "local_lock")
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		panic(errors.NewUnknownf("failed to create lock directory: %s, error: %w", dir, err))
	}

	filePath := path.Join(dir, fmt.Sprintf("%x", sha1.Sum([]byte(resource))))
	fd, err := syscall.Open(filePath, syscall.O_CREAT|syscall.O_RDWR, 0666)
	if err != nil {
		panic(errors.NewUnknownf("failed to open lock file: %s, error: %w", filePath, err))
	}

	return &LocalLock{
		BaseDistributedLock: BaseDistributedLock{
			resource: resource,
			id:       uuid.NewString(),
			ttl:      ttl,
		},
		path: filePath,
		fd:   fd,
		mu:   &sync.Mutex{},
	}
}

func (l *LocalLock) Lock(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	err := syscall.Flock(l.fd, syscall.LOCK_EX)
	if err != nil {
		return errors.NewUnknownf("failed to lock file: %s, error: %w", l.path, err)
	}
	l.doLock(ctx)
	log.GetLoggerFromCtx(ctx).Debugf("Lock aquired: %s, lockID: %s, expiration: %s", l.resource, l.id, l.expiration)
	return nil
}

func (l *LocalLock) doLock(ctx context.Context) {
	l.expiration = time.Now().Add(l.ttl)
	l.locked = true

	go func() {
		for {
			if !l.locked {
				return
			}
			if l.expiration.After(time.Now()) {
				time.Sleep(l.ttl / 2)
				continue
			}
			err := l.Unlock(ctx)
			if err != nil {
				log.GetLoggerFromCtx(ctx).Errorf("Failed to unlock file: %s, error: %s", l.path, err)
			}
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

	err := syscall.Flock(l.fd, syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			logger.Debugf("Lock not acquired: %s, lockID: %s", l.resource, l.id)
			return false, nil
		}
		return false, errors.NewUnknownf("failed to lock: %s file: %s, error: %w", l.resource, l.path, err)
	}
	l.doLock(ctx)
	log.GetLoggerFromCtx(ctx).Debugf("Lock aquired: %s, lockID: %s, expiration: %s", l.resource, l.id, l.expiration)
	return true, nil
}

func (l *LocalLock) Unlock(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	err := syscall.Flock(l.fd, syscall.LOCK_UN)
	if err != nil {
		return errors.NewUnknownf("failed to unlock file: %s, error: %w", l.path, err)
	}

	if l.autoExtenderCancel != nil {
		l.autoExtenderCancel(context.Canceled)
		l.autoExtenderCancel = nil
	}

	l.extendedCount = 0
	l.expiration = time.Time{}
	l.locked = false

	log.GetLoggerFromCtx(ctx).Debugf("Lock unlocked: %s, lockID: %s, file: %s", l.resource, l.id, l.path)
	return nil
}

func (l *LocalLock) Extend(ctx context.Context) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.locked {
		return false, nil
	}

	if time.Now().After(l.expiration) {
		// Unlock will eventually happen in the background
		return false, nil
	}

	l.extendedCount++
	l.expiration = time.Now().Add(l.ttl)
	log.GetLoggerFromCtx(ctx).Tracef(
		"Lock extended: %s, lockID: %s, expiration: %s, extendedCount: %d",
		l.resource,
		l.id,
		l.expiration,
		l.extendedCount,
	)
	return true, nil
}

func (l *LocalLock) AutoExtend(ctx context.Context) (context.Context, error) {
	return autoExtend(ctx, l, &l.BaseDistributedLock)
}
