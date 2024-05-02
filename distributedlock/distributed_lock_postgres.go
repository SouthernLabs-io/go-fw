package distributedlock

import (
	"context"
	"database/sql"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	lib "github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
	fwsync "github.com/southernlabs-io/go-fw/sync"
)

var errSchemaAlreadyInitialized = errors.Newf("SCHEMA_ALREADY_INITIALIZED", "schema already initialized by another instance")

type DistributedPostgresLock struct {
	BaseDistributedLock
}

var _ DistributedLock = &DistributedPostgresLock{}

func NewDistributedPostgresLock(resource string, ttl time.Duration) *DistributedPostgresLock {
	return &DistributedPostgresLock{
		BaseDistributedLock{
			resource: resource,
			id:       uuid.NewString(),
			ttl:      ttl,
		},
	}
}

func setupDB(tx *lib.DBTx) error {
	err := tx.Exec(`
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
	tx, _ := lib.WithTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	// we wrap the deferred call so it is not bound to this tx in case we have to re create
	// the tx due to schema initialization race
	defer func() {
		tx.DeferredCommitOrRollback(&err)
	}()
	var until time.Time
	var currLockID string

	err = setupDB(tx)
	if err != nil {
		if errors.Is(err, errSchemaAlreadyInitialized) {
			lib.GetLoggerFromCtx(ctx).Debug("Another instance has already initialized the distributed_lock schema")
			// The transaction is dead. We need to roll it back and create a new one
			if err = tx.Rollback().Error; err != nil {
				return false, err
			}
			tx, _ = lib.WithTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
		} else {
			return false, err
		}
	}

	err = tx.Raw(
		"SELECT instance_id FROM distributed_lock.lock WHERE resource = ? FOR UPDATE",
		l.resource,
	).Scan(&currLockID).Error
	if err != nil {
		return false, err
	}

	if currLockID != "" {
		err = tx.Raw(
			`UPDATE distributed_lock.lock SET instance_id = ?, expiration = now() + INTERVAL '1 second' * ?
		 			WHERE resource = ? AND instance_id = ? AND expiration < now() RETURNING expiration`,
			l.id,
			l.ttl.Seconds(),
			l.resource,
			currLockID,
		).Scan(&until).Error
	} else {
		err = tx.Raw(
			`INSERT INTO distributed_lock.lock (resource, instance_id, expiration)
					VALUES(?, ?, now() + INTERVAL '1 second' * ?)
					ON CONFLICT DO NOTHING RETURNING expiration`,
			l.resource,
			l.id,
			l.ttl.Seconds(),
		).Scan(&until).Error
	}
	if err != nil {
		return false, err
	}

	logger := lib.GetLoggerFromCtx(ctx)
	if !until.IsZero() {
		l.expiration = until
		logger.Debugf("acquired: %s, lockID: %s, expiration: %s", l.resource, l.id, until)
		return true, nil
	}

	logger.Debugf("not acquired: %s, lockID: %s", l.resource, l.id)
	return false, nil
}

func (l *DistributedPostgresLock) Unlock(ctx context.Context) error {
	res := lib.InTx(ctx).Exec(
		"UPDATE distributed_lock.lock SET expiration = now() WHERE resource = ? AND instance_id = ? AND expiration > now()",
		l.resource,
		l.id,
	)
	if res.Error != nil {
		return res.Error
	}

	if l.autoExtenderCancel != nil {
		l.autoExtenderCancel(context.Canceled)
		l.autoExtenderCancel = nil
	}
	l.expiration = time.Time{}
	l.extendedCount = 0
	logger := lib.GetLoggerFromCtx(ctx)
	if res.RowsAffected != 0 {
		logger.Debugf("unlocked: %s, lockID: %s", l.resource, l.id)
	} else {
		logger.Debugf("it was already expired or unlocked: %s, lockID: %s", l.resource, l.id)
	}
	return nil
}

func (l *DistributedPostgresLock) Extend(ctx context.Context) (bool, error) {
	var until time.Time
	var extendedCount int
	err := lib.InTx(ctx).Raw(
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

	logger := lib.GetLoggerFromCtx(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			l.expiration = time.Time{}
			logger.Debugf("not extended: %s, lockID: %s, expiration: %s", l.resource, l.id, l.expiration)
			return false, nil
		}
		return false, err
	}

	l.expiration = until
	l.extendedCount = extendedCount
	logger.Debugf(
		"extended: %s, lockID: %s, expiration: %s, extendedCount: %d",
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
