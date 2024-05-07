package distributedlock

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/redis"
	fwsync "github.com/southernlabs-io/go-fw/sync"
)

type RedisFactory struct {
	rds redis.Redis
}

func NewRedisFactory(rds redis.Redis) *RedisFactory {
	return &RedisFactory{rds: rds}
}

func (f *RedisFactory) NewDistributedLock(resource string, ttl time.Duration) DistributedLock {
	return NewDistributedRedisLock(f.rds, resource, ttl)
}

type DistributedRedisLock struct {
	BaseDistributedLock

	redis redis.Redis
}

var _ DistributedLock = &DistributedRedisLock{}

func NewDistributedRedisLock(redis redis.Redis, resource string, ttl time.Duration) *DistributedRedisLock {
	return &DistributedRedisLock{
		BaseDistributedLock: BaseDistributedLock{
			resource: resource,
			id:       uuid.NewString(),
			ttl:      ttl,
		},
		redis: redis,
	}
}

func (l *DistributedRedisLock) extendedCountKey() string {
	return l.resource + "::" + l.id + "::extended_count"
}

// Lock will block until the lock is acquired or an error occurs.
func (l *DistributedRedisLock) Lock(ctx context.Context) error {
	var locked bool
	var err error
	for {
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

var setNXAndExtendedCountScript = `
	if redis.call("SET", KEYS[1], ARGV[1], "NX", "PX", ARGV[2]) then
		redis.call("SET", KEYS[2], "0", "PX", ARGV[2])
		return 1
	else
		return 0
	end
`

// TryLock will attempt to acquire the lock and return true if successful.
func (l *DistributedRedisLock) TryLock(ctx context.Context) (bool, error) {
	logger := core.GetLoggerFromCtx(ctx)
	rdb := l.redis.Client

	setInt, err := rdb.Eval(ctx, setNXAndExtendedCountScript, []string{l.resource, l.extendedCountKey()}, l.id, l.ttl.Milliseconds()).Result()

	if err != nil {
		return false, err
	}
	set := setInt.(int64) != 0

	if set {
		l.extendedCount = 0
		// Lua scripts can't return more than one value, so we pull the ttl in milliseconds separately.
		pttl := rdb.PTTL(ctx, l.resource).Val()
		l.expiration = time.Now().Add(pttl)
		logger.Debugf("Lock aquired: %s, lockID: %s, expiration: %s", l.resource, l.id, l.expiration)
	} else {
		logger.Debugf("Lock not aquired: %s, lockID: %s", l.resource, l.id)
	}

	return set, nil
}

var deleteKeysScript = `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1], KEYS[2])
	else
		return 0
	end
`

func (l *DistributedRedisLock) Unlock(ctx context.Context) error {
	logger := core.GetLoggerFromCtx(ctx)
	rdb := l.redis.Client

	deletedCount, err := rdb.Eval(ctx, deleteKeysScript, []string{l.resource, l.extendedCountKey()}, l.id).Result()
	if err != nil {
		return err
	}

	if l.autoExtenderCancel != nil {
		l.autoExtenderCancel(context.Canceled)
		l.autoExtenderCancel = nil
	}

	unlocked := deletedCount.(int64) != 0

	if unlocked {
		logger.Debugf("lock unlocked: %s, lockID: %s", l.resource, l.id)
	} else {
		logger.Debugf("lock not aquired and unlocked: %s, lockID: %s", l.resource, l.id)
	}

	l.extendedCount = 0
	l.expiration = time.Time{}

	return nil
}

var extendSetsAndIncrExtendedCountScript = `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		if redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2]) then
			redis.call("EXPIRE", KEYS[2], ARGV[2])
			return redis.call("INCR", KEYS[2])
		else
			return 0
		end
	else
		return 0
	end
`

func (l *DistributedRedisLock) Extend(ctx context.Context) (bool, error) {
	logger := core.GetLoggerFromCtx(ctx)
	rdb := l.redis.Client

	extendedCount, err := rdb.Eval(ctx, extendSetsAndIncrExtendedCountScript, []string{l.resource, l.extendedCountKey()}, l.id, l.ttl.Milliseconds()).Result()
	if err != nil {
		return false, err
	}

	// We can assume if the extendedCount is 0, then the lock was not extended.
	set := extendedCount.(int64) != 0

	if set {
		l.extendedCount = int(extendedCount.(int64))
		// Lua scripts can't return more than one value, so we pull the ttl in milliseconds separately.
		pttl := rdb.PTTL(ctx, l.resource).Val()
		l.expiration = time.Now().Add(pttl)
		logger.Tracef(
			"Lock extended: %s, lockID: %s, expiration: %s, extendedCount: %d",
			l.resource,
			l.id,
			l.expiration,
			l.extendedCount,
		)
	} else {
		l.expiration = time.Time{}
		l.extendedCount = 0
		logger.Warnf("Lock not extended: %s, lockID: %s, expiration: %s", l.resource, l.id, l.expiration)
	}

	return set, nil
}

func (l *DistributedRedisLock) AutoExtend(ctx context.Context) (context.Context, error) {
	return autoExtend(ctx, l, &l.BaseDistributedLock)
}
