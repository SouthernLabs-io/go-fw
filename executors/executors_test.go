package executors_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/executors"
	"github.com/southernlabs-io/go-fw/test"
)

func TestDefaultExecutor_Submit(t *testing.T) {
	logger := test.GetTestLogger(t)
	ctx := context.Background()
	exec := executors.NewDefaultExecutor(ctx, 1, 1)
	require.NotNil(t, exec)
	require.EqualValues(t, 1, exec.Concurrency())
	require.EqualValues(t, 1, exec.QueueCapacity())
	require.EqualValues(t, 0, exec.QueueLength())

	// Submit a task
	future, err := exec.Submit(func(ctx context.Context) error {
		logger.Debug("run")
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, future)
	require.False(t, future.Canceled())

	// Wait for the task to be done
	<-future.Await()
	require.True(t, future.Done())
	require.NoError(t, future.Err())
	require.False(t, future.Cancel())
	require.False(t, future.Canceled())
	require.EqualValues(t, 0, exec.QueueLength())

	// Submit a task that returns an error
	future, err = exec.Submit(func(ctx context.Context) error {
		logger.Debug("run 2")
		return errors.Newf("TASK_ERROR", "run 2")
	})
	require.NoError(t, err)
	require.NotNil(t, future)
	require.False(t, future.Canceled())

	// Wait for the task to be done
	<-future.Await()
	require.True(t, future.Done())
	require.Error(t, future.Err())
	require.True(t, errors.IsCode(future.Err(), "TASK_ERROR"))
	require.False(t, future.Cancel())
	require.False(t, future.Canceled())
	require.EqualValues(t, 0, exec.QueueLength())

	// Submit a task that panics, it should be wrapped in an error
	future, err = exec.Submit(func(ctx context.Context) error {
		logger.Debug("run 3")
		panic("run 3")
	})
	require.NoError(t, err)
	require.NotNil(t, future)
	require.False(t, future.Canceled())

	// Wait for the task to be done
	<-future.Await()
	require.True(t, future.Done())
	require.Error(t, future.Err())
	require.True(t, errors.IsCode(future.Err(), executors.ErrCodeTaskPanic))

	// Test cancelling a task by submitting a long-running one
	future, err = exec.Submit(func(ctx context.Context) error {
		logger.Debug("run 4")
		time.Sleep(time.Hour)
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, future)
	require.False(t, future.Done())
	require.False(t, future.Canceled())
	require.Eventually(t, func() bool { return exec.QueueLength() == 0 }, time.Second, time.Millisecond)

	// This task will never run
	future, err = exec.Submit(func(ctx context.Context) error {
		logger.Debug("run 5")
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, future)
	require.False(t, future.Done())
	require.False(t, future.Canceled())
	require.Eventually(t, func() bool { return exec.QueueLength() == 1 }, time.Second, time.Millisecond)

	// Check submitting a new task fails with ErrExecutorQueueFull
	nilFuture, err := exec.Submit(func(ctx context.Context) error {
		logger.Debug("run 6")
		return nil
	})
	require.Error(t, err)
	require.ErrorIs(t, err, executors.ErrExecutorQueueFull)
	require.Nil(t, nilFuture)

	// Cancel the task to remove it from the queue
	require.True(t, future.Cancel())
	require.True(t, future.Canceled())
	require.Eventually(t, func() bool { <-future.Await(); return true }, time.Second, 1)
	require.True(t, future.Done())
	require.Error(t, future.Err())
	require.ErrorIs(t, future.Err(), context.Canceled)
	require.Eventually(t, func() bool { return exec.QueueLength() == 0 }, time.Second, time.Millisecond)

	// Cancel the executor will never terminate
	require.True(t, exec.Cancel())
	require.False(t, exec.Cancel())
	require.True(t, exec.Canceled())
	require.Never(t, func() bool { return exec.AwaitTermination(time.Millisecond * 100) }, time.Second/2, 1)
	require.False(t, exec.Terminated())
}

func TestDefaultExecutor_SubmitProducer(t *testing.T) {
	logger := test.GetTestLogger(t)
	ctx := context.Background()
	executor := executors.NewDefaultExecutor(ctx, 1, 0)
	require.NotNil(t, executor)
	require.EqualValues(t, 1, executor.Concurrency())
	require.EqualValues(t, 0, executor.QueueCapacity())
	require.EqualValues(t, 0, executor.QueueLength())

	// Submit a producer task
	future, err := executor.SubmitProducer(func(ctx context.Context) (any, error) {
		logger.Debug("run")
		return "run", nil
	})
	require.NoError(t, err)
	require.NotNil(t, future)
	require.Eventually(t, func() bool { <-future.Await(); return true }, time.Second, 1)
	require.NoError(t, future.Err())
	require.False(t, future.Cancel())
	require.False(t, future.Canceled())
	require.EqualValues(t, "run", future.Value())

	// Submit a producer task that returns an error
	future, err = executor.SubmitProducer(func(ctx context.Context) (any, error) {
		logger.Debug("run 2")
		return nil, errors.Newf("PRODUCER_ERROR", "run 2")
	})
	require.NoError(t, err)
	require.NotNil(t, future)
	require.Eventually(t, func() bool { <-future.Await(); return true }, time.Second, 1)
	require.Error(t, future.Err())
	require.True(t, errors.IsCode(future.Err(), "PRODUCER_ERROR"))
	require.False(t, future.Cancel())
	require.False(t, future.Canceled())
	require.Nil(t, future.Value())

	// Cancel the executor
	require.True(t, executor.Cancel())
	require.False(t, executor.Cancel())
	require.True(t, executor.Canceled())
	require.Eventually(t, func() bool { return executor.AwaitTermination(time.Millisecond * 100) }, time.Second, 1)
	require.True(t, executor.Terminated())
}

func TestDefaultExecutor_Schedule(t *testing.T) {
	logger := test.GetTestLogger(t)
	ctx := context.Background()
	exec := executors.NewDefaultExecutor(ctx, 1, 0)
	require.NotNil(t, exec)
	require.EqualValues(t, 1, exec.Concurrency())
	require.EqualValues(t, 0, exec.QueueCapacity())
	require.EqualValues(t, 0, exec.QueueLength())

	start := time.Now()
	delay := time.Millisecond * 100
	future, err := exec.Schedule(func(ctx context.Context) error {
		logger.Debugf("run: %s", time.Since(start))
		require.GreaterOrEqual(t, time.Since(start), delay)
		return nil
	}, delay)
	require.NoError(t, err)
	require.NotNil(t, future)
	require.False(t, future.Done())
	require.False(t, future.Periodic())
	require.Eventually(t, func() bool { return exec.QueueLength() == 1 }, time.Second, time.Millisecond)
	require.Greater(t, future.Delay(), time.Duration(0))
	require.LessOrEqual(t, future.Delay(), delay)

	// Check scheduling a new task fails with ErrExecutorQueueFull
	nilFuture, err := exec.Schedule(func(ctx context.Context) error {
		logger.Debug("run 2")
		return nil
	}, delay)
	require.Error(t, err)
	require.ErrorIs(t, err, executors.ErrExecutorQueueFull)
	require.Nil(t, nilFuture)

	// Wait for the task to complete
	require.Eventually(t, func() bool { <-future.Await(); return true }, delay*2, 1)
	require.True(t, future.Done())
	require.NoError(t, future.Err())
	require.False(t, future.Cancel())
	require.False(t, future.Canceled())
	require.Eventually(t, func() bool { return exec.QueueLength() == 0 }, time.Second, time.Millisecond)

	// Cancel the executor
	require.True(t, exec.Cancel())
	require.False(t, exec.Cancel())
	require.True(t, exec.Canceled())
	require.Eventually(t, func() bool { return exec.AwaitTermination(time.Millisecond * 100) }, time.Second, 1)
	require.True(t, exec.Terminated())
}

func TestDefaultExecutor_ScheduleWithFixedDelay(t *testing.T) {
	logger := test.GetTestLogger(t)
	ctx := context.Background()
	exec := executors.NewDefaultExecutor(ctx, 1, 0)
	require.NotNil(t, exec)
	require.EqualValues(t, 1, exec.Concurrency())
	require.EqualValues(t, 0, exec.QueueCapacity())
	require.EqualValues(t, 0, exec.QueueLength())

	start := atomic.Value{}
	start.Store(time.Now())
	count := atomic.Int32{}
	initialDelay := time.Millisecond * 50
	delay := initialDelay * 2
	future, err := exec.ScheduleWithFixedDelay(func(ctx context.Context) error {
		st := start.Load().(time.Time)
		logger.Debugf("run: %s", time.Since(st))
		if count.Load() == 0 {
			require.GreaterOrEqual(t, time.Since(st), initialDelay)
		} else {
			require.GreaterOrEqual(t, time.Since(st), delay)
		}
		start.Store(time.Now())
		count.Add(1)
		return nil
	}, initialDelay, delay)
	require.NoError(t, err)
	require.NotNil(t, future)
	require.True(t, future.Periodic())
	require.False(t, future.Done())
	require.Eventually(t, func() bool { return exec.QueueLength() == 1 }, time.Second, time.Millisecond)
	require.Greater(t, future.Delay(), time.Duration(0))
	require.LessOrEqual(t, future.Delay(), delay)

	// Check scheduling a new task fails with ErrExecutorQueueFull
	nilFuture, err := exec.ScheduleWithFixedDelay(func(ctx context.Context) error {
		logger.Debug("run 2")
		return nil
	}, initialDelay, delay)
	require.Error(t, err)
	require.ErrorIs(t, err, executors.ErrExecutorQueueFull)
	require.Nil(t, nilFuture)

	expectedCount := 5
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		logger.Debugf("count: %d", count.Load())
		count := count.Load()
		assert.EqualValues(c, expectedCount, count)
	}, initialDelay+delay*time.Duration(expectedCount+1), delay)
	require.Never(t, func() bool { <-future.Await(); return true }, delay*2, 1)
	require.False(t, future.Done())
	require.NoError(t, future.Err())

	// Cancel the task to stop it from running again
	require.True(t, future.Cancel())
	require.False(t, future.Cancel())
	require.True(t, future.Canceled())
	require.Eventually(t, func() bool { <-future.Await(); return true }, time.Second, 1)
	require.True(t, future.Done())
	require.Error(t, future.Err())
	require.ErrorIs(t, future.Err(), context.Canceled)
	require.Eventually(t, func() bool { return exec.QueueLength() == 0 }, time.Second, time.Millisecond)

	// Cancel the executor
	require.True(t, exec.Cancel())
	require.False(t, exec.Cancel())
	require.True(t, exec.Canceled())
	require.Eventually(t, func() bool { return exec.AwaitTermination(time.Millisecond * 100) }, time.Second, 1)
	require.True(t, exec.Terminated())
}

func TestDefaultExecutor_ScheduleAtFixedRate(t *testing.T) {
	logger := test.GetTestLogger(t)
	ctx := context.Background()
	exec := executors.NewDefaultExecutor(ctx, 1, 0)
	require.NotNil(t, exec)
	require.EqualValues(t, 1, exec.Concurrency())
	require.EqualValues(t, 0, exec.QueueCapacity())
	require.EqualValues(t, 0, exec.QueueLength())

	// We only modify them from a single goroutine, so we don't need to use atomic
	start := time.Now()
	count := 0
	initialDelay := time.Millisecond * 50
	period := initialDelay * 2
	future, err := exec.ScheduleWithFixedRate(func(ctx context.Context) error {
		if count == 0 {
			require.GreaterOrEqual(t, time.Since(start), initialDelay)
			start = start.Add(initialDelay)
		} else {
			require.GreaterOrEqual(t, time.Since(start), period)
			start = start.Add(period)
		}

		count += 1
		return nil
	}, initialDelay, period)
	require.NoError(t, err)
	require.NotNil(t, future)
	require.Eventually(t, func() bool { return exec.QueueLength() == 1 }, time.Second, time.Millisecond)
	require.Greater(t, future.Delay(), time.Duration(0))
	require.LessOrEqual(t, future.Delay(), period)

	// Check scheduling a new task fails with ErrExecutorQueueFull
	nilFuture, err := exec.ScheduleWithFixedRate(func(ctx context.Context) error {
		logger.Debug("run 6")
		return nil
	}, initialDelay, period)
	require.Error(t, err)
	require.ErrorIs(t, err, executors.ErrExecutorQueueFull)
	require.Nil(t, nilFuture)

	expectedCount := 5
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		logger.Debugf("count: %d", count)
		assert.EqualValues(c, expectedCount, count)
	}, initialDelay+period*time.Duration(expectedCount+1), period)
	require.Never(t, func() bool {
		<-future.Await()
		return true
	}, period*2, 1)
	require.NoError(t, future.Err())

	// Cancel the task to stop it from running
	require.True(t, future.Cancel())
	require.True(t, future.Canceled())
	require.Error(t, future.Err())
	require.ErrorIs(t, future.Err(), context.Canceled)
	require.Eventually(t, func() bool { return exec.QueueLength() == 0 }, time.Second, time.Millisecond)

	// Cancel the executor
	require.True(t, exec.Cancel())
	require.False(t, exec.Cancel())
	require.True(t, exec.Canceled())
	require.Eventually(t, func() bool { return exec.AwaitTermination(time.Millisecond * 100) }, time.Second, 1)
	require.True(t, exec.Terminated())
}

func TestDefaultExecutor_ScheduleAtFixedRateWithSlowTask(t *testing.T) {
	logger := test.GetTestLogger(t)
	ctx := context.Background()
	executor := executors.NewDefaultExecutor(ctx, 1, 1)
	require.NotNil(t, executor)
	require.EqualValues(t, 1, executor.Concurrency())
	require.EqualValues(t, 1, executor.QueueCapacity())
	require.EqualValues(t, 0, executor.QueueLength())

	start := atomic.Value{}
	start.Store(time.Now())
	count := atomic.Int32{}
	initialDelay := time.Millisecond * 50
	period := initialDelay * 2
	slowFactor := time.Duration(2)
	future, err := executor.ScheduleWithFixedRate(func(ctx context.Context) error {
		st := start.Load().(time.Time)
		if count.Load() == 0 {
			require.GreaterOrEqual(t, time.Since(st), initialDelay)
			start.Store(st.Add(initialDelay))
		} else {
			require.GreaterOrEqual(t, time.Since(st), period)
			start.Store(st.Add(period))
		}
		count.Add(1)
		logger.Debugf("count: %d, sleeping now", count.Load())
		time.Sleep(period * slowFactor)
		logger.Debugf("count: %d, sleeping done", count.Load())
		return nil
	}, initialDelay, period)
	require.NoError(t, err)
	require.NotNil(t, future)
	require.Eventually(t, func() bool { return executor.QueueLength() == 1 }, time.Second, time.Millisecond)
	require.Greater(t, future.Delay(), time.Duration(0))
	require.LessOrEqual(t, future.Delay(), period)
	expectedCount := 5
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.EqualValues(c, expectedCount, count.Load())
	}, initialDelay+period*slowFactor*time.Duration(expectedCount+1), period)
	require.Never(t, func() bool { <-future.Await(); return true }, period*2, 1)
	require.NoError(t, future.Err())
	require.False(t, future.Done())

	// Cancel the task to stop it from running
	require.True(t, future.Cancel())
	require.True(t, future.Canceled())
	require.Eventually(t, func() bool { <-future.Await(); return true }, 2*time.Second, 1)
	require.Error(t, future.Err())
	require.ErrorIs(t, future.Err(), context.Canceled)
	require.Eventually(t, func() bool { return executor.QueueLength() == 0 }, time.Second, time.Millisecond)

	// Cancel the executor
	require.True(t, executor.Cancel())
	require.False(t, executor.Cancel())
	require.True(t, executor.Canceled())
	require.Eventually(t, func() bool { return executor.AwaitTermination(time.Millisecond * 100) }, time.Second, 1)
	require.True(t, executor.Terminated())
}

func TestDefaultExecutor_CancelTask(t *testing.T) {
	logger := test.GetTestLogger(t)
	ctx := context.Background()
	executor := executors.NewDefaultExecutor(ctx, 1, 1)
	require.NotNil(t, executor)
	require.EqualValues(t, 1, executor.Concurrency())
	require.EqualValues(t, 1, executor.QueueCapacity())
	require.EqualValues(t, 0, executor.QueueLength())

	// Submit cancellable task
	running := atomic.Bool{}
	future, err := executor.Submit(func(ctx context.Context) error {
		logger.Debug("run")
		running.Store(true)
		<-ctx.Done()
		// Sleep to show a slow cancelling task
		time.Sleep(time.Millisecond * 10)
		return ctx.Err()
	})
	require.NoError(t, err)
	require.NotNil(t, future)
	require.False(t, future.Done())
	require.False(t, future.Canceled())
	require.Eventually(t, func() bool { return running.Load() }, time.Second, time.Millisecond)

	// Cancel the task
	require.True(t, future.Cancel())
	require.False(t, future.Cancel())
	require.True(t, future.Canceled())
	require.False(t, future.Done())
	require.Eventually(t, func() bool { <-future.Await(); return true }, time.Second, 1)
	require.True(t, future.Done())
	require.Error(t, future.Err())
	require.ErrorIs(t, future.Err(), context.Canceled)

	// Cancel the executor
	require.True(t, executor.Cancel())
	require.False(t, executor.Cancel())
	require.True(t, executor.Canceled())
	require.Eventually(t, func() bool { return executor.AwaitTermination(time.Millisecond * 100) }, time.Second, 1)
	require.True(t, executor.Terminated())
}

func TestDefaultExecutor_Cancel(t *testing.T) {
	logger := test.GetTestLogger(t)
	ctx := context.Background()
	exec := executors.NewDefaultExecutor(ctx, 1, 100)
	require.NotNil(t, exec)
	require.EqualValues(t, 1, exec.Concurrency())
	require.EqualValues(t, 100, exec.QueueCapacity())
	require.EqualValues(t, 0, exec.QueueLength())

	running := atomic.Bool{}
	longRunningFuture, err := exec.Submit(func(ctx context.Context) error {
		logger.Debug("run")
		running.Store(true)
		for running.Load() {
			time.Sleep(time.Millisecond * 10)
		}
		logger.Debug("run done")
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, longRunningFuture)
	require.Eventually(t, func() bool { return running.Load() }, time.Millisecond*100, time.Millisecond)

	future, err := exec.Submit(func(ctx context.Context) error {
		logger.Debug("run 2")
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, future)
	require.Eventually(t, func() bool { return exec.QueueLength() == 1 }, time.Millisecond*100, time.Millisecond)

	scheduledFuture, err := exec.Schedule(func(ctx context.Context) error {
		logger.Debug("run 3")
		return nil
	}, time.Millisecond*10)
	require.NoError(t, err)
	require.NotNil(t, scheduledFuture)
	require.Eventually(t, func() bool { return exec.QueueLength() == 2 }, time.Millisecond*100, time.Millisecond)
	time.Sleep(scheduledFuture.Delay())
	require.Eventually(t, func() bool { return exec.QueueLength() == 2 }, time.Millisecond*100, time.Millisecond)
	require.LessOrEqual(t, scheduledFuture.Delay(), time.Duration(0))

	// Cancel the executor
	require.False(t, exec.Canceled())
	require.False(t, exec.Terminated())
	require.True(t, exec.Cancel())
	require.False(t, exec.Cancel())
	require.True(t, exec.Canceled())
	require.False(t, exec.Terminated())
	require.EqualValues(t, 2, exec.QueueLength())

	// Check scheduling a new task fails with ErrExecutorCanceled
	nilFuture, err := exec.Schedule(func(ctx context.Context) error {
		logger.Debug("run 4")
		return nil
	}, time.Second)
	require.Error(t, err)
	require.ErrorIs(t, err, executors.ErrExecutorCanceled)
	require.Nil(t, nilFuture)

	// CancelNow should return all tasks
	tasks := exec.CancelNow()
	require.EqualValues(t, 2, len(tasks))
	require.EqualValues(t, 0, exec.QueueLength())

	// CancelNow again should return an empty list
	require.Empty(t, exec.CancelNow())

	// Await should fail with timeout
	require.Never(t, func() bool { return exec.AwaitTermination(time.Millisecond * 10) }, time.Millisecond*100, 1)

	// Instruct the task to stop running
	running.Store(false)
	require.Eventually(t, func() bool { return exec.AwaitTermination(time.Millisecond * 10) }, time.Second/2, 1)
	require.True(t, exec.Terminated())

	// Check scheduling a new task fails with ErrExecutorCanceled after termination
	nilFuture, err = exec.Schedule(func(ctx context.Context) error {
		logger.Debug("run 5")
		return nil
	}, time.Second)
	require.Error(t, err)
	require.ErrorIs(t, err, executors.ErrExecutorCanceled)
	require.Nil(t, nilFuture)
}

func TestDefaultExecutor_ContextCanceled(t *testing.T) {
	logger := test.GetTestLogger(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	exec := executors.NewDefaultExecutor(ctx, 1, 100)
	require.NotNil(t, exec)

	// Submit a task and check it was executed
	future, err := exec.Submit(func(ctx context.Context) error {
		logger.Debug("run")
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, future)
	require.Eventually(t, func() bool { <-future.Await(); return true }, time.Second, 1)
	require.NoError(t, future.Err())

	// Cancel de ctx should close the executor
	require.False(t, exec.Canceled())
	require.False(t, exec.Terminated())
	cancel()
	require.Eventually(t, func() bool { return exec.Canceled() }, time.Second, time.Millisecond)
	require.Eventually(t, func() bool { return exec.Terminated() }, time.Second, time.Millisecond)
}

func TestDefaultExecutor_CapacityAndConcurrency(t *testing.T) {
	logger := test.GetTestLogger(t)
	ctx := context.Background()

	// Start with a constrained executor
	concurrency := 0
	capacity := 0
	exec := executors.NewDefaultExecutor(ctx, concurrency, capacity)
	require.NotNil(t, exec)
	require.EqualValues(t, concurrency, exec.Concurrency())
	require.EqualValues(t, capacity, exec.QueueCapacity())
	require.EqualValues(t, 0, exec.QueueLength())

	// Submitting a task should fail with ErrExecutorQueueFull
	_, err := exec.Submit(func(ctx context.Context) error {
		logger.Debug("run")
		return nil
	})
	require.Error(t, err)
	require.ErrorIs(t, err, executors.ErrExecutorQueueFull)

	// Scheduling a task should fail with ErrExecutorQueueFull
	_, err = exec.Schedule(func(ctx context.Context) error {
		logger.Debug("run 2")
		return nil
	}, time.Second)
	require.Error(t, err)
	require.ErrorIs(t, err, executors.ErrExecutorQueueFull)

	// Increase concurrency so tasks can be executed
	concurrency = 10
	exec.SetConcurrency(concurrency)
	require.EqualValues(t, concurrency, exec.Concurrency())

	// Submit concurrency long running tasks, all of them should start execution
	started := atomic.Int32{}
	for i := 0; i < concurrency; i++ {
		_, err := exec.Submit(func(ctx context.Context) error {
			started.Add(1)
			time.Sleep(time.Hour)
			return nil
		})
		require.NoError(t, err)
	}

	// Wait till all tasks are executing
	require.Eventually(t, func() bool { return started.Load() == int32(concurrency) }, time.Second, time.Millisecond)
	require.EqualValues(t, 0, exec.QueueLength())

	// Submit one more task should fail with ErrExecutorQueueFull
	_, err = exec.Submit(func(ctx context.Context) error {
		started.Add(1)
		return nil
	})
	require.Error(t, err)
	require.ErrorIs(t, err, executors.ErrExecutorQueueFull)

	// Increase capacity so a task can be queued
	capacity += 1
	exec.SetQueueCapacity(capacity)
	require.EqualValues(t, capacity, exec.QueueCapacity())

	// Submit again
	_, err = exec.Submit(func(ctx context.Context) error {
		started.Add(1)
		time.Sleep(time.Hour)
		return nil
	})
	require.NoError(t, err)

	// Wait till the task shows in the queue
	require.Eventually(t, func() bool { return exec.QueueLength() == 1 }, time.Second, time.Millisecond)
	require.EqualValues(t, concurrency, started.Load())

	// Increase concurrency so the task is started
	concurrency += 1
	exec.SetConcurrency(concurrency)
	require.EqualValues(t, concurrency, exec.Concurrency())

	// Wait till the task is started
	require.Eventually(t, func() bool { return exec.QueueLength() == 0 }, time.Second/2, time.Millisecond)
	require.EqualValues(t, concurrency, started.Load())

	// Cancel the executor
	require.True(t, exec.Cancel())
	require.False(t, exec.Cancel())
	require.True(t, exec.Canceled())

	// Long running tasks will never finish
	require.Never(t, func() bool { return exec.AwaitTermination(time.Millisecond * 100) }, time.Second/2, 1)
	require.False(t, exec.Terminated())
}

func TestDefaultExecutor_Unbound(t *testing.T) {
	ctx := context.Background()

	// Start with zero concurrency and capacity unbounded
	exec := executors.NewDefaultExecutor(ctx, 0, -1)
	require.NotNil(t, exec)
	require.EqualValues(t, 0, exec.Concurrency())
	require.EqualValues(t, -1, exec.QueueCapacity())
	require.EqualValues(t, 0, exec.QueueLength())

	// Submit a large number of simple tasks
	largeNumber := 1_000
	futures := make([]executors.Future, largeNumber)
	for i := 0; i < largeNumber; i++ {
		var err error
		futures[i], err = exec.Submit(func(ctx context.Context) error {
			return nil
		})
		require.NoError(t, err)
		require.NotNil(t, futures[i])
	}

	// Check all tasks are in the queue
	require.Eventually(t, func() bool { return exec.QueueLength() == largeNumber }, time.Second, time.Millisecond)

	// Change the concurrency to Unbounded
	exec.SetConcurrency(-1)
	require.EqualValues(t, -1, exec.Concurrency())

	// Change the capacity to zero
	exec.SetQueueCapacity(0)
	require.EqualValues(t, 0, exec.QueueCapacity())

	// Submitting a new task should not fail because concurrency is unbounded
	future, err := exec.Submit(func(ctx context.Context) error {
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, future)
	futures = append(futures, future)

	// Wait till all tasks are completed
	for _, future = range futures {
		require.Eventually(t, func() bool { <-future.Await(); return true }, time.Second, 1)
		require.NoError(t, future.Err())
	}
	require.EqualValues(t, 0, exec.QueueLength())

	// Cancel the executor
	require.True(t, exec.Cancel())
	require.False(t, exec.Cancel())
	require.True(t, exec.Canceled())
	require.Eventually(t, func() bool { return exec.AwaitTermination(time.Millisecond * 100) }, time.Second, 1)
	require.True(t, exec.Terminated())
}

func BenchmarkTest10K(b *testing.B) {
	b.StopTimer()
	total := 10_000

	// Create as many executors as needed
	executorsList := make([]*executors.DefaultExecutor, b.N)
	for i := 0; i < b.N; i++ {
		executorsList[i] = executors.NewDefaultExecutor(context.Background(), 100, -1)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		executor := executorsList[i]
		for j := 0; j < total; j++ {
			_, err := executor.Submit(func(context.Context) error {
				return nil
			})
			if err != nil {
				b.Fatal("task failed to be submitted at: ", j)
			}
		}
		executor.Cancel()
		if !executor.AwaitTermination(time.Second * 3) {
			b.Fatal("executor still working: ", executor.QueueLength())
		}
	}
}
