package distributedlock

import (
	"context"
	"database/sql"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/driver/pgdriver"

	database "github.com/southernlabs-io/go-fw/database/bun"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
	fwsync "github.com/southernlabs-io/go-fw/sync"
)

type PostgresBunFactory struct{}

func NewPostgresBunFactory() *PostgresBunFactory {
	return &PostgresBunFactory{}
}

func (f *PostgresBunFactory) NewDistributedLock(resource string, ttl time.Duration) DistributedLock {
	return NewDistributedPostgresBunLock(resource, ttl)
}

type DistributedPostgresBunLock struct {
	BaseDistributedLock
}

var _ DistributedLock = &DistributedPostgresBunLock{}

func NewDistributedPostgresBunLock(resource string, ttl time.Duration) *DistributedPostgresBunLock {
	return &DistributedPostgresBunLock{
		BaseDistributedLock{
			resource: resource,
			id:       uuid.NewString(),
			ttl:      ttl,
		},
	}
}

func setupDBBun(ctx context.Context, idb bun.IDB) error {
	_, err := idb.NewRaw(`
		CREATE SCHEMA IF NOT EXISTS distributed_lock;
		CREATE TABLE IF NOT EXISTS distributed_lock.lock (
			resource TEXT PRIMARY KEY,
			instance_id TEXT NOT NULL,
			expiration TIMESTAMP WITH TIME ZONE NOT NULL,
			first_locked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
			extended_count INTEGER NOT NULL DEFAULT 0
		)`).Exec(ctx)
	if err == nil {
		return nil
	}
	var pgErr pgdriver.Error
	if errors.As(err, &pgErr) && pgErr.Field('n') == "pg_namespace_nspname_index" {
		return errSchemaAlreadyInitialized
	}
	return err
}

func (l *DistributedPostgresBunLock) Lock(ctx context.Context) error {
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
		// Sleep with jitter between 10% and 20% of ttl
		err = fwsync.Sleep(ctx, l.ttl/10+time.Duration(rand.Int63n(int64(l.ttl)/10)))
		if err != nil {
			return err
		}
	}
}

func (l *DistributedPostgresBunLock) TryLock(ctx context.Context) (locked bool, err error) {
	idb := database.GetDBFromCtx(ctx)
	var until time.Time

	err = setupDBBun(ctx, idb)
	if err != nil {
		if errors.Is(err, errSchemaAlreadyInitialized) {
			log.GetLoggerFromCtx(ctx).Debug("Another instance has already initialized the distributed_lock schema")
		} else {
			return false, err
		}
	}

	err = idb.NewRaw(
		`INSERT INTO distributed_lock.lock (resource, instance_id, expiration, extended_count)
VALUES (?, ?, clock_timestamp() + INTERVAL '1 second' * ?, 0)
ON CONFLICT (resource) DO UPDATE
SET instance_id = EXCLUDED.instance_id,
    expiration = EXCLUDED.expiration,
    extended_count = 0
WHERE distributed_lock.lock.expiration < clock_timestamp()
RETURNING expiration`,
		l.resource,
		l.id,
		l.ttl.Seconds(),
	).Scan(ctx, &until)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return false, errors.NewUnknownf("failed to acquire lock, error: %w", err)
		}
	}

	logger := log.GetLoggerFromCtx(ctx)
	if !until.IsZero() {
		l.setExpiration(until)
		logger.Debugf("Lock acquired: %s, lockID: %s, expiration: %s", l.resource, l.id, until)
		return true, nil
	}

	logger.Debugf("Lock not acquired: %s, lockID: %s", l.resource, l.id)
	return false, nil
}

func (l *DistributedPostgresBunLock) Unlock(ctx context.Context) error {
	res, err := database.GetDBFromCtx(ctx).NewRaw(
		"UPDATE distributed_lock.lock SET expiration = clock_timestamp() WHERE resource = ? AND instance_id = ? AND expiration > clock_timestamp()",
		l.resource,
		l.id,
	).Exec(ctx)
	if err != nil {
		return errors.NewUnknownf("failed to unlock: %w", err)
	}

	l.cancelAndResetAutoExtender(context.Canceled)
	l.resetLockState()
	logger := log.GetLoggerFromCtx(ctx)
	rows, err := res.RowsAffected()
	if err != nil {
		return errors.NewUnknownf("failed to get rows affected on unlock: %w", err)
	}
	if rows != 0 {
		logger.Debugf("Lock unlocked: %s, lockID: %s", l.resource, l.id)
	} else {
		logger.Debugf("it was already expired or unlocked: %s, lockID: %s", l.resource, l.id)
	}
	return nil
}

func (l *DistributedPostgresBunLock) Extend(ctx context.Context) (bool, error) {
	var until time.Time
	var extendedCount int
	err := database.GetDBFromCtx(ctx).NewRaw(
		`UPDATE distributed_lock.lock
				SET expiration = clock_timestamp() + INTERVAL '1 second' * ?,
				    extended_count = extended_count + 1 
                WHERE resource = ?
	                  AND instance_id = ?
	                  AND expiration > clock_timestamp()
                RETURNING expiration, extended_count`,
		l.ttl.Seconds(),
		l.resource,
		l.id,
	).Scan(ctx, &until, &extendedCount)

	logger := log.GetLoggerFromCtx(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			expiration := l.Expiration()
			l.resetLockState()
			logger.Warnf("Lock not extended: %s, lockID: %s, expiration: %s", l.resource, l.id, expiration)
			return false, nil
		}
		return false, err
	}

	l.setLockState(until, extendedCount)
	logger.Tracef(
		"Lock extended: %s, lockID: %s, expiration: %s, extendedCount: %d",
		l.resource,
		l.id,
		until,
		extendedCount,
	)
	return true, nil
}

func (l *DistributedPostgresBunLock) AutoExtend(ctx context.Context) (context.Context, error) {
	return autoExtend(ctx, l, &l.BaseDistributedLock)
}
