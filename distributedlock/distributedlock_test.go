package distributedlock_test

import (
	"context"
	"sync"
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

func setupDBBun(t *testing.T) (ctx context.Context) {
	t.Parallel()

	test.FxIntegrationWithDBBun(t).Populate(&ctx)
	return
}

func setupRedis(t *testing.T) (rds redis.Redis, ctx context.Context) {
	t.Parallel()

	test.FxIntegration(t, test.FxExportRedis).Populate(&rds, &ctx)
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
		ctx := setupDBBun(t)
		dLock := distributedlock.NewDistributedPostgresBunLock("myResource_"+uuid.NewString(), ttl)
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
		ctx := setupDBBun(t)
		dLock := distributedlock.NewDistributedPostgresBunLock("myResource_"+uuid.NewString(), ttl)
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
		ctx := setupDBBun(t)
		name := "myResource_" + uuid.NewString()
		dLock1 := distributedlock.NewDistributedPostgresBunLock(name, ttl)
		dLock2 := distributedlock.NewDistributedPostgresBunLock(name, ttl)
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
		ctx := setupDBBun(t)
		dLock1 := distributedlock.NewDistributedPostgresBunLock("myResource1_"+uuid.NewString(), ttl)
		dLock2 := distributedlock.NewDistributedPostgresBunLock("myResource2_"+uuid.NewString(), ttl)
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
		ctx := setupDBBun(t)
		dLock := distributedlock.NewDistributedPostgresBunLock("myResource_"+uuid.NewString(), ttl)
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

	// Check it was extended at least once
	extended := dLock.ExtendedCount()
	require.GreaterOrEqual(t, extended, 1)
	require.False(t, called.Load())

	// Cancel the auto extender parent context
	cCtxCancel(cancelErr)

	// Wait for one cycle
	time.Sleep(time.Until(dLock.Expiration()) + 1)

	// Check it was not extended, or at least one more
	require.True(t, dLock.ExtendedCount() <= extended+1)

	// Unlock
	err = dLock.Unlock(ctx)
	require.NoError(t, err)
}

func TestAutoExtenderStopWhenUnlocked(t *testing.T) {
	ttl := time.Second * 2
	t.Run("Postgres", func(t *testing.T) {
		ctx := setupDBBun(t)
		dLock := distributedlock.NewDistributedPostgresBunLock("myResource_"+uuid.NewString(), ttl)
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

	// Check it was extended at least once
	require.GreaterOrEqual(t, dLock.ExtendedCount(), 1)

	// Unlock
	err = dLock.Unlock(ctx)
	require.NoError(t, err)

	// Check auto extender stopped
	require.ErrorIs(t, context.Cause(aeCtx), context.Canceled)
}

func TestConcurrentLockAcquisition(t *testing.T) {
	ttl := time.Second * 5
	t.Run("Postgres", func(t *testing.T) {
		ctx := setupDBBun(t)
		resource := "concurrent_" + uuid.NewString()
		factory := func() distributedlock.DistributedLock {
			return distributedlock.NewDistributedPostgresBunLock(resource, ttl)
		}
		testConcurrentLockAcquisition(t, ctx, factory)
	})
	t.Run("Redis", func(t *testing.T) {
		rds, ctx := setupRedis(t)
		resource := "concurrent_" + uuid.NewString()
		factory := func() distributedlock.DistributedLock {
			return distributedlock.NewDistributedRedisLock(rds, resource, ttl)
		}
		testConcurrentLockAcquisition(t, ctx, factory)
	})
	t.Run("Local", func(t *testing.T) {
		ctx := setupLocal(t)
		resource := "concurrent_" + uuid.NewString()
		factory := func() distributedlock.DistributedLock {
			return distributedlock.NewDistributedLocalLock(resource, ttl)
		}
		testConcurrentLockAcquisition(t, ctx, factory)
	})
}

func testConcurrentLockAcquisition(
	t *testing.T,
	ctx context.Context,
	factory func() distributedlock.DistributedLock,
) {
	const numGoroutines = 20
	successCount := atomic.Int32{}
	failCount := atomic.Int32{}
	startBarrier := make(chan struct{})

	// Launch all goroutines to wait at the barrier
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			<-startBarrier // Wait for signal to start simultaneously

			dLock := factory()
			locked, err := dLock.TryLock(ctx)
			require.NoError(t, err, "goroutine %d got error", id)

			if locked {
				successCount.Add(1)
				// Hold lock briefly
				time.Sleep(50 * time.Millisecond)
				err = dLock.Unlock(ctx)
				require.NoError(t, err, "goroutine %d failed to unlock", id)
			} else {
				failCount.Add(1)
			}
		}(i)
	}

	// Give goroutines time to reach the barrier
	time.Sleep(100 * time.Millisecond)

	// Release all goroutines simultaneously
	close(startBarrier)

	// Wait for all to complete
	time.Sleep(time.Second * 2)

	// Exactly ONE should have succeeded
	require.Equal(t, int32(1), successCount.Load(), "expected exactly 1 lock acquisition")
	require.Equal(t, int32(numGoroutines-1), failCount.Load(), "expected %d failures", numGoroutines-1)
}

func TestConcurrentLockWithBlocking(t *testing.T) {
	ttl := time.Second * 2
	t.Run("Postgres", func(t *testing.T) {
		ctx := setupDBBun(t)
		resource := "blocking_" + uuid.NewString()
		factory := func() distributedlock.DistributedLock {
			return distributedlock.NewDistributedPostgresBunLock(resource, ttl)
		}
		testConcurrentLockWithBlocking(t, ctx, factory)
	})
	t.Run("Redis", func(t *testing.T) {
		rds, ctx := setupRedis(t)
		resource := "blocking_" + uuid.NewString()
		factory := func() distributedlock.DistributedLock {
			return distributedlock.NewDistributedRedisLock(rds, resource, ttl)
		}
		testConcurrentLockWithBlocking(t, ctx, factory)
	})
	t.Run("Local", func(t *testing.T) {
		ctx := setupLocal(t)
		resource := "blocking_" + uuid.NewString()
		factory := func() distributedlock.DistributedLock {
			return distributedlock.NewDistributedLocalLock(resource, ttl)
		}
		testConcurrentLockWithBlocking(t, ctx, factory)
	})
}

func testConcurrentLockWithBlocking(
	t *testing.T,
	ctx context.Context,
	factory func() distributedlock.DistributedLock,
) {
	const numGoroutines = 5
	successCount := atomic.Int32{}
	startBarrier := make(chan struct{})
	var wg sync.WaitGroup

	// Each goroutine will call Lock() which blocks until it can acquire
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-startBarrier

			dLock := factory()

			// Lock() blocks until acquired
			err := dLock.Lock(ctx)
			require.NoError(t, err, "goroutine %d failed to lock", id)

			successCount.Add(1)

			// Hold lock briefly
			time.Sleep(100 * time.Millisecond)

			err = dLock.Unlock(ctx)
			require.NoError(t, err, "goroutine %d failed to unlock", id)
		}(i)
	}

	// Give goroutines time to reach the barrier
	time.Sleep(50 * time.Millisecond)

	// Release all goroutines simultaneously
	close(startBarrier)

	// Wait for all goroutines to complete with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines completed successfully
		require.Equal(t, int32(numGoroutines), successCount.Load())
	case <-time.After(15 * time.Second):
		t.Fatalf("timeout waiting for locks, only %d/%d acquired", successCount.Load(), numGoroutines)
	}
}

func TestConcurrentExtend(t *testing.T) {
	ttl := time.Second * 2
	t.Run("Postgres", func(t *testing.T) {
		ctx := setupDBBun(t)
		resource := "extend_" + uuid.NewString()
		dLock := distributedlock.NewDistributedPostgresBunLock(resource, ttl)
		testConcurrentExtend(t, ctx, dLock)
	})
	t.Run("Redis", func(t *testing.T) {
		rds, ctx := setupRedis(t)
		resource := "extend_" + uuid.NewString()
		dLock := distributedlock.NewDistributedRedisLock(rds, resource, ttl)
		testConcurrentExtend(t, ctx, dLock)
	})
	t.Run("Local", func(t *testing.T) {
		ctx := setupLocal(t)
		resource := "extend_" + uuid.NewString()
		dLock := distributedlock.NewDistributedLocalLock(resource, ttl)
		testConcurrentExtend(t, ctx, dLock)
	})
}

func testConcurrentExtend(
	t *testing.T,
	ctx context.Context,
	dLock distributedlock.DistributedLock,
) {
	// Acquire the lock
	err := dLock.Lock(ctx)
	require.NoError(t, err)

	const numExtenders = 10
	successCount := atomic.Int32{}
	startBarrier := make(chan struct{})

	// Multiple goroutines try to extend simultaneously
	for i := 0; i < numExtenders; i++ {
		go func(id int) {
			<-startBarrier

			extended, err := dLock.Extend(ctx)
			require.NoError(t, err, "goroutine %d got error during extend", id)

			if extended {
				successCount.Add(1)
			}
		}(i)
	}

	// Give goroutines time to reach the barrier
	time.Sleep(50 * time.Millisecond)

	// Release all goroutines simultaneously
	close(startBarrier)

	// Wait for all to complete
	time.Sleep(500 * time.Millisecond)

	// At least one should have succeeded (may be more due to timing)
	require.GreaterOrEqual(t, successCount.Load(), int32(1), "at least one extend should succeed")

	// Cleanup
	err = dLock.Unlock(ctx)
	require.NoError(t, err)
}

func TestConcurrentUnlock(t *testing.T) {
	ttl := time.Second * 3
	t.Run("Postgres", func(t *testing.T) {
		ctx := setupDBBun(t)
		resource := "unlock_" + uuid.NewString()
		dLock := distributedlock.NewDistributedPostgresBunLock(resource, ttl)
		testConcurrentUnlock(t, ctx, dLock)
	})
	t.Run("Redis", func(t *testing.T) {
		rds, ctx := setupRedis(t)
		resource := "unlock_" + uuid.NewString()
		dLock := distributedlock.NewDistributedRedisLock(rds, resource, ttl)
		testConcurrentUnlock(t, ctx, dLock)
	})
	t.Run("Local", func(t *testing.T) {
		ctx := setupLocal(t)
		resource := "unlock_" + uuid.NewString()
		dLock := distributedlock.NewDistributedLocalLock(resource, ttl)
		testConcurrentUnlock(t, ctx, dLock)
	})
}

func testConcurrentUnlock(
	t *testing.T,
	ctx context.Context,
	dLock distributedlock.DistributedLock,
) {
	// Acquire the lock
	err := dLock.Lock(ctx)
	require.NoError(t, err)

	const numUnlockers = 5
	startBarrier := make(chan struct{})
	errorCount := atomic.Int32{}

	// Multiple goroutines try to unlock simultaneously
	for i := 0; i < numUnlockers; i++ {
		go func(id int) {
			<-startBarrier

			err := dLock.Unlock(ctx)
			// Unlock should not fail, it just won't affect anything if already unlocked
			if err != nil {
				errorCount.Add(1)
			}
		}(i)
	}

	// Give goroutines time to reach the barrier
	time.Sleep(50 * time.Millisecond)

	// Release all goroutines simultaneously
	close(startBarrier)

	// Wait for all to complete
	time.Sleep(500 * time.Millisecond)

	// No errors should occur
	require.Equal(t, int32(0), errorCount.Load(), "concurrent unlocks should not error")
}

func TestRaceConditionExpiredLock(t *testing.T) {
	ttl := time.Millisecond * 500 // Short TTL to ensure expiration
	t.Run("Postgres", func(t *testing.T) {
		ctx := setupDBBun(t)
		resource := "expired_" + uuid.NewString()
		factory := func() distributedlock.DistributedLock {
			return distributedlock.NewDistributedPostgresBunLock(resource, ttl)
		}
		testRaceConditionExpiredLock(t, ctx, factory)
	})
	t.Run("Redis", func(t *testing.T) {
		rds, ctx := setupRedis(t)
		resource := "expired_" + uuid.NewString()
		factory := func() distributedlock.DistributedLock {
			return distributedlock.NewDistributedRedisLock(rds, resource, ttl)
		}
		testRaceConditionExpiredLock(t, ctx, factory)
	})
	t.Run("Local", func(t *testing.T) {
		ctx := setupLocal(t)
		resource := "expired_" + uuid.NewString()
		factory := func() distributedlock.DistributedLock {
			return distributedlock.NewDistributedLocalLock(resource, ttl)
		}
		testRaceConditionExpiredLock(t, ctx, factory)
	})
}

func testRaceConditionExpiredLock(
	t *testing.T,
	ctx context.Context,
	factory func() distributedlock.DistributedLock,
) {
	// First goroutine acquires the lock
	dLock1 := factory()
	err := dLock1.Lock(ctx)
	require.NoError(t, err)

	lockTTL := dLock1.TTL()

	// Wait for lock to expire
	time.Sleep(lockTTL + 100*time.Millisecond)

	const numCompetitors = 10
	successCount := atomic.Int32{}
	startBarrier := make(chan struct{})

	// Multiple goroutines try to acquire the expired lock simultaneously
	for i := 0; i < numCompetitors; i++ {
		go func(id int) {
			<-startBarrier

			dLock := factory()
			locked, err := dLock.TryLock(ctx)
			require.NoError(t, err, "goroutine %d got error", id)

			if locked {
				successCount.Add(1)
				time.Sleep(50 * time.Millisecond)
				dLock.Unlock(ctx)
			}
		}(i)
	}

	// Give goroutines time to reach the barrier
	time.Sleep(50 * time.Millisecond)

	// Release all goroutines simultaneously
	close(startBarrier)

	// Wait for completion
	time.Sleep(time.Second)

	// Exactly ONE should have succeeded in acquiring the expired lock
	require.Equal(t, int32(1), successCount.Load(), "exactly one should acquire expired lock")
}
