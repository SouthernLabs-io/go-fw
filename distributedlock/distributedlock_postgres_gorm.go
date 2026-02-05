package distributedlock

import (
	"context"
	"database/sql"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	database "github.com/southernlabs-io/go-fw/database/gorm"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
	fwsync "github.com/southernlabs-io/go-fw/sync"
)

var errSchemaAlreadyInitialized = errors.Newf("SCHEMA_ALREADY_INITIALIZED", "schema already initialized by another instance")

type PostgresGORMFactory struct{}

func NewPostgresGORMFactory() *PostgresGORMFactory {
	return &PostgresGORMFactory{}
}

func (f *PostgresGORMFactory) NewDistributedLock(resource string, ttl time.Duration) DistributedLock {
	return NewDistributedPostgresGORMLock(resource, ttl)
}

type DistributedPostgresLock struct {
	BaseDistributedLock
}

var _ DistributedLock = &DistributedPostgresLock{}

func NewDistributedPostgresGORMLock(resource string, ttl time.Duration) *DistributedPostgresLock {
	return &DistributedPostgresLock{
		BaseDistributedLock{
			resource: resource,
			id:       uuid.NewString(),
			ttl:      ttl,
		},
	}
}

func setupDBGORM(db *database.DBTx) error {
	err := db.Exec(`
		CREATE SCHEMA IF NOT EXISTS distributed_lock;
		CREATE TABLE IF NOT EXISTS distributed_lock.lock (
			resource TEXT PRIMARY KEY,
			instance_id TEXT NOT NULL,
			expiration TIMESTAMP WITH TIME ZONE NOT NULL,
			first_locked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
			extended_count INTEGER NOT NULL DEFAULT 0
		)`).Error
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == "pg_namespace_nspname_index" {
		return errSchemaAlreadyInitialized
	}
	return err
}

func (l *DistributedPostgresLock) Lock(ctx context.Context) error {
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

func (l *DistributedPostgresLock) TryLock(ctx context.Context) (locked bool, err error) {
	db := database.CurrentTx(ctx)
	var until time.Time

	err = setupDBGORM(db)
	if err != nil {
		if errors.Is(err, errSchemaAlreadyInitialized) {
			log.GetLoggerFromCtx(ctx).Debug("Another instance has already initialized the distributed_lock schema")
		} else {
			return false, err
		}
	}

	err = db.Raw(
		`INSERT INTO distributed_lock.lock (resource, instance_id, expiration, extended_count)
VALUES (?, ?, now() + INTERVAL '1 second' * ?, 0)
ON CONFLICT (resource) DO UPDATE
SET instance_id = EXCLUDED.instance_id,
    expiration = EXCLUDED.expiration,
    extended_count = 0
WHERE distributed_lock.lock.expiration < now()
RETURNING expiration`,
		l.resource,
		l.id,
		l.ttl.Seconds(),
	).Scan(&until).Error
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

func (l *DistributedPostgresLock) Unlock(ctx context.Context) error {
	res := database.CurrentTx(ctx).Exec(
		"UPDATE distributed_lock.lock SET expiration = now() WHERE resource = ? AND instance_id = ? AND expiration > now()",
		l.resource,
		l.id,
	)
	if res.Error != nil {
		return res.Error
	}

	l.cancelAndResetAutoExtender(context.Canceled)
	l.resetLockState()
	logger := log.GetLoggerFromCtx(ctx)
	if res.RowsAffected != 0 {
		logger.Debugf("Lock unlocked: %s, lockID: %s", l.resource, l.id)
	} else {
		logger.Debugf("it was already expired or unlocked: %s, lockID: %s", l.resource, l.id)
	}
	return nil
}

func (l *DistributedPostgresLock) Extend(ctx context.Context) (bool, error) {
	var until time.Time
	var extendedCount int
	err := database.CurrentTx(ctx).Raw(
		`UPDATE distributed_lock.lock
				SET expiration = now() + INTERVAL '1 second' * ?,
				    extended_count = extended_count + 1 
                WHERE resource = ?
                  AND instance_id = ?
                  AND expiration > now()
                RETURNING expiration, extended_count`,
		l.ttl.Seconds(),
		l.resource,
		l.id,
	).Row().Scan(&until, &extendedCount)

	logger := log.GetLoggerFromCtx(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.resetLockState()
			logger.Warnf("Lock not extended: %s, lockID: %s, expiration: %s", l.resource, l.id, time.Time{})
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

func (l *DistributedPostgresLock) AutoExtend(ctx context.Context) (context.Context, error) {
	return autoExtend(ctx, l, &l.BaseDistributedLock)
}
