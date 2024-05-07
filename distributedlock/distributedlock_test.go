package distributedlock_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/distributedlock"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/redis"
	"github.com/southernlabs-io/go-fw/test"
)

func setupDB(t *testing.T) (ctx context.Context) {
	t.Parallel()

	test.FxIntegrationWithDB(t).Populate(&ctx)
	return
}

func setupRedis(t *testing.T) (rds redis.Redis, ctx context.Context) {
	t.Parallel()

	test.FxIntegration(t, test.ModuleRedis).Populate(&rds, &ctx)
	return
}

func setupLocal(t *testing.T) (ctx context.Context) {
	t.Parallel()

	test.FxUnit(t).Populate(&ctx)
	return
}

func TestLockOneTimeUse(t *testing.T) {
	ttl := time.Second * 2
	t.Run("Postgres", func(t *testing.T) {
		ctx := setupDB(t)
		dLock := distributedlock.NewDistributedPostgresLock("myResource_"+uuid.NewString(), ttl)
		testLockOneTimeUse(t, ctx, dLock)
	})

	t.Run("Redis", func(t *testing.T) {
		rds, ctx := setupRedis(t)
		dLock := distributedlock.NewDistributedRedisLock(rds, "myResource_"+uuid.NewString(), ttl)
		testLockOneTimeUse(t, ctx, dLock)
	})

	t.Run("Local", func(t *testing.T) {
		ctx := setupLocal(t)
		dLock := distributedlock.NewDistributedLocalLock("myResource_"+uuid.NewString(), ttl)
		testLockOneTimeUse(t, ctx, dLock)
	})
}
func testLockOneTimeUse(t *testing.T, ctx context.Context, dLock distributedlock.DistributedLock) {
	require.NotNil(t, dLock)
	require.Zero(t, dLock.Expiration())

	// First lock should succeed
	err := dLock.Lock(ctx)
	require.NoError(t, err)
	require.NotZero(t, dLock.Expiration())
	require.Greater(t, dLock.Expiration().Unix(), time.Now().Unix())

	// TryLock should fail as it is already locked
	locked, err := dLock.TryLock(ctx)
	require.NoError(t, err)
	require.False(t, locked)

	// Second lock will block until the previous lock expires. This is the behavior of sync.Mutex
	err = dLock.Lock(ctx)
	require.NoError(t, err)

	// Unlock
	err = dLock.Unlock(ctx)
	require.NoError(t, err)
	require.Zero(t, dLock.Expiration())

	// TryLock should succeed
	locked, err = dLock.TryLock(ctx)
	require.NoError(t, err)
	require.True(t, locked)

	// TryLock again should fail as it is already locked
	locked, err = dLock.TryLock(ctx)
	require.NoError(t, err)
	require.False(t, locked)

	// Unlock
	err = dLock.Unlock(ctx)
	require.NoError(t, err)
	require.Zero(t, dLock.Expiration())
}

func TestLongRunningWorker(t *testing.T) {
	ttl := time.Second * 2
	t.Run("Postgres", func(t *testing.T) {
		ctx := setupDB(t)
		dLock := distributedlock.NewDistributedPostgresLock("myResource_"+uuid.NewString(), ttl)
		testLongRunningWorker(t, ctx, dLock)
	})
	t.Run("Redis", func(t *testing.T) {
		rds, ctx := setupRedis(t)
		dLock := distributedlock.NewDistributedRedisLock(rds, "myResource_"+uuid.NewString(), ttl)
		testLongRunningWorker(t, ctx, dLock)
	})

	t.Run("Local", func(t *testing.T) {
		ctx := setupLocal(t)
		dLock := distributedlock.NewDistributedLocalLock("myResource_"+uuid.NewString(), ttl)
		testLongRunningWorker(t, ctx, dLock)
	})
}

func testLongRunningWorker(t *testing.T, ctx context.Context, dLock distributedlock.DistributedLock) {
	require.NotNil(t, dLock)
	require.Zero(t, dLock.Expiration())

	err := dLock.Lock(ctx)
	require.NoError(t, err)
	require.NotZero(t, dLock.Expiration())

	// Iterate 10 times
	var extendedCount int
	for i := 0; i < 10; i++ {
		if time.Now().Add(1 * time.Second).After(dLock.Expiration()) {
			// Lock will expire in less than 1 seconds, extend it
			prevUntil := dLock.Expiration()
			extended, err := dLock.Extend(ctx)
			require.NoError(t, err)
			require.True(t, extended)
			require.Greater(t, dLock.Expiration().Unix(), prevUntil.Unix())
			extendedCount++
			require.EqualValues(t, extendedCount, dLock.ExtendedCount())
		}
		time.Sleep(time.Millisecond * 500)
	}
	require.EqualValues(t, 4, extendedCount)

	// Unlock
	err = dLock.Unlock(ctx)
	require.NoError(t, err)
	require.Zero(t, dLock.Expiration())

	// Extend should not succeed when it is not locked
	extended, err := dLock.Extend(ctx)
	require.NoError(t, err)
	require.False(t, extended)
	require.Zero(t, dLock.Expiration())
}

func TestMultipleAccessToSameResource(t *testing.T) {
	ttl := time.Second * 3
	t.Run("Postgres", func(t *testing.T) {
		ctx := setupDB(t)
		name := "myResource_" + uuid.NewString()
		dLock1 := distributedlock.NewDistributedPostgresLock(name, ttl)
		dLock2 := distributedlock.NewDistributedPostgresLock(name, ttl)
		testMultipleAccessToSameResource(t, ctx, dLock1, dLock2)
	})
	t.Run("Redis", func(t *testing.T) {
		rds, ctx := setupRedis(t)
		name := "myResource_" + uuid.NewString()
		dLock1 := distributedlock.NewDistributedRedisLock(rds, name, ttl)
		dLock2 := distributedlock.NewDistributedRedisLock(rds, name, ttl)
		testMultipleAccessToSameResource(t, ctx, dLock1, dLock2)
	})
	t.Run("Local", func(t *testing.T) {
		ctx := setupLocal(t)
		dLock1 := distributedlock.NewDistributedLocalLock("myResource", ttl)
		dLock2 := distributedlock.NewDistributedLocalLock("myResource", ttl)
		testMultipleAccessToSameResource(t, ctx, dLock1, dLock2)
	})
}

func testMultipleAccessToSameResource(
	t *testing.T,
	ctx context.Context,
	dLock1 distributedlock.DistributedLock,
	dLock2 distributedlock.DistributedLock,
) {
	require.NotNil(t, dLock1)
	require.Zero(t, dLock1.Expiration())

	require.NotNil(t, dLock2)
	require.Zero(t, dLock2.Expiration())

	// Test two locks on the same resource cannot be acquired at the same time
	set, err := dLock1.TryLock(ctx)
	require.NoError(t, err)
	require.Equal(t, true, set)
	set, err = dLock2.TryLock(ctx)
	require.NoError(t, err)
	require.Equal(t, false, set)

}

func TestMultipleResources(t *testing.T) {
	ttl := time.Second * 3
	t.Run("Postgres", func(t *testing.T) {
		ctx := setupDB(t)
		dLock1 := distributedlock.NewDistributedPostgresLock("myResource1_"+uuid.NewString(), ttl)
		dLock2 := distributedlock.NewDistributedPostgresLock("myResource2_"+uuid.NewString(), ttl)
		testMultipleResources(t, ctx, dLock1, dLock2)
	})
	t.Run("Redis", func(t *testing.T) {
		rds, ctx := setupRedis(t)
		dLock1 := distributedlock.NewDistributedRedisLock(rds, "myResource1_"+uuid.NewString(), ttl)
		dLock2 := distributedlock.NewDistributedRedisLock(rds, "myResource2_"+uuid.NewString(), ttl)
		testMultipleResources(t, ctx, dLock1, dLock2)
	})
	t.Run("Local", func(t *testing.T) {
		ctx := setupLocal(t)
		dLock1 := distributedlock.NewDistributedLocalLock("myResource1_"+uuid.NewString(), ttl)
		dLock2 := distributedlock.NewDistributedLocalLock("myResource2_"+uuid.NewString(), ttl)
		testMultipleResources(t, ctx, dLock1, dLock2)
	})
}

func testMultipleResources(
	t *testing.T,
	ctx context.Context,
	dLock1 distributedlock.DistributedLock,
	dLock2 distributedlock.DistributedLock,
) {
	require.NotNil(t, dLock1)
	require.Zero(t, dLock1.Expiration())

	require.NotNil(t, dLock2)
	require.Zero(t, dLock2.Expiration())

	// Lock dLock1
	locked, err := dLock1.TryLock(ctx)
	require.NoError(t, err)
	require.True(t, locked)
	require.NotZero(t, dLock1.Expiration())
	require.Greater(t, dLock1.Expiration().Unix(), time.Now().Unix())

	// Lock dLock2
	locked, err = dLock2.TryLock(ctx)
	require.NoError(t, err)
	require.True(t, locked)
	require.NotZero(t, dLock2.Expiration())
	require.Greater(t, dLock2.Expiration().Unix(), time.Now().Unix())

	// Unlock dLock1, dLock2 should still be locked
	err = dLock1.Unlock(ctx)
	require.NoError(t, err)
	require.Zero(t, dLock1.Expiration())
	locked, err = dLock2.TryLock(ctx)
	require.NoError(t, err)
	require.False(t, locked)
	require.NotZero(t, dLock2.Expiration())
	require.Greater(t, dLock2.Expiration().Unix(), time.Now().Unix())

	// Lock dLock1 again, dLock2 should still be locked
	locked, err = dLock1.TryLock(ctx)
	require.NoError(t, err)
	require.True(t, locked)
	require.NotZero(t, dLock1.Expiration())
	require.Greater(t, dLock1.Expiration().Unix(), time.Now().Unix())
	locked, err = dLock2.TryLock(ctx)
	require.NoError(t, err)
	require.False(t, locked)
	require.NotZero(t, dLock2.Expiration())
	require.Greater(t, dLock2.Expiration().Unix(), time.Now().Unix())

	// Wait until both expire
	time.Sleep(time.Until(dLock1.Expiration()))
	time.Sleep(time.Until(dLock2.Expiration()))

	// Try extend
	extended, err := dLock1.Extend(ctx)
	require.NoError(t, err)
	require.False(t, extended)
	extended, err = dLock2.Extend(ctx)
	require.NoError(t, err)
	require.False(t, extended)

	// Unlock should not fail when it is not locked, this is different from sync.Mutex that panics in this case
	err = dLock1.Unlock(ctx)
	require.NoError(t, err)
	err = dLock2.Unlock(ctx)
	require.NoError(t, err)
}

func TestAutoExtender(t *testing.T) {
	ttl := time.Second * 2
	t.Run("Postgres", func(t *testing.T) {
		ctx := setupDB(t)
		dLock := distributedlock.NewDistributedPostgresLock("myResource_"+uuid.NewString(), ttl)
		testAutoExtender(t, ctx, dLock)
	})
	t.Run("Redis", func(t *testing.T) {
		rds, ctx := setupRedis(t)
		dLock := distributedlock.NewDistributedRedisLock(rds, "myResource_"+uuid.NewString(), ttl)
		testAutoExtender(t, ctx, dLock)
	})
	t.Run("Local", func(t *testing.T) {
		ctx := setupLocal(t)
		dLock := distributedlock.NewDistributedLocalLock("myResource_"+uuid.NewString(), ttl)
		testAutoExtender(t, ctx, dLock)
	})
}

func testAutoExtender(t *testing.T, ctx context.Context, dLock distributedlock.DistributedLock) {
	require.NotNil(t, dLock)
	require.Zero(t, dLock.Expiration())

	// Call AutoExtend when it is not locked should fail
	aeCtx, err := dLock.AutoExtend(ctx)
	require.Error(t, err)
	require.True(t, errors.IsCode(err, errors.ErrCodeBadState))
	require.Nil(t, aeCtx)

	// Lock
	err = dLock.Lock(ctx)
	require.NoError(t, err)
	require.NotZero(t, dLock.Expiration())

	// Call AutoExtend when it is locked should succeed
	cCtx, cCtxCancel := context.WithCancelCause(ctx)
	aeCtx, err = dLock.AutoExtend(cCtx)
	require.NoError(t, err)
	require.NotNil(t, aeCtx)
	called := atomic.Bool{}
	cancelErr := errors.Newf("CANCEL", "cancel")
	go func() {
		<-aeCtx.Done()
		require.ErrorIs(t, context.Cause(aeCtx), cancelErr)
		called.Store(true)
	}()

	// Wait for one cycle
	time.Sleep(time.Until(dLock.Expiration()) + 1)

	// Check it was extended once
	require.Equal(t, 1, dLock.ExtendedCount())
	require.False(t, called.Load())

	// Cancel the auto extender parent context
	cCtxCancel(cancelErr)

	// Wait for one cycle
	time.Sleep(time.Until(dLock.Expiration()) + 1)

	// Check it was not extended, or at least one more
	require.True(t, dLock.ExtendedCount() <= 2)

	// Unlock
	err = dLock.Unlock(ctx)
	require.NoError(t, err)
}

func TestAutoExtenderStopWhenUnlocked(t *testing.T) {
	ttl := time.Second * 2
	t.Run("Postgres", func(t *testing.T) {
		ctx := setupDB(t)
		dLock := distributedlock.NewDistributedPostgresLock("myResource_"+uuid.NewString(), ttl)
		testAutoExtenderStopWhenUnlocked(t, ctx, dLock)
	})
	t.Run("Redis", func(t *testing.T) {
		rds, ctx := setupRedis(t)
		dLock := distributedlock.NewDistributedRedisLock(rds, "myResource_"+uuid.NewString(), ttl)
		testAutoExtenderStopWhenUnlocked(t, ctx, dLock)
	})
	t.Run("Local", func(t *testing.T) {
		ctx := setupLocal(t)
		dLock := distributedlock.NewDistributedLocalLock("myResource_"+uuid.NewString(), ttl)
		testAutoExtenderStopWhenUnlocked(t, ctx, dLock)
	})
}

func testAutoExtenderStopWhenUnlocked(t *testing.T, ctx context.Context, dLock distributedlock.DistributedLock) {
	require.NotNil(t, dLock)
	require.Zero(t, dLock.Expiration())

	err := dLock.Lock(ctx)
	require.NoError(t, err)

	// AutoExtend
	aeCtx, err := dLock.AutoExtend(ctx)
	require.NoError(t, err)
	require.NotNil(t, aeCtx)

	// Wait for one cycle
	time.Sleep(time.Until(dLock.Expiration()) + 1)

	// Check it was extended once
	require.Equal(t, 1, dLock.ExtendedCount())

	// Unlock
	err = dLock.Unlock(ctx)
	require.NoError(t, err)

	// Check auto extender stopped
	require.ErrorIs(t, context.Cause(aeCtx), context.Canceled)
}
