package executors

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/errors"
)

func TestTask_Once_NormalFlow(t *testing.T) {
	task := newTaskOnce(t, 0)

	// Move to Running
	require.True(t, task.setRunning())
	require.False(t, task.setRunning())
	require.False(t, task.Done())
	require.False(t, task.Canceled())

	// Move to Done
	require.True(t, task.setDone(nil, nil))
	require.False(t, task.setDone(nil, nil))
	require.Eventually(t, func() bool { <-task.Await(); return true }, time.Second, 1)
	require.True(t, task.Done())
	require.False(t, task.Canceled())
	require.False(t, task.Cancel())
}

func TestTask_Periodic_NormalFlow(t *testing.T) {
	for _, task := range []*_Task{
		newTaskFixedRate(t, time.Millisecond*10),
		// Delay must be greater than the used in all previous task
		newTaskFixedDelay(t, time.Millisecond*30),
	} {
		t.Run(fmt.Sprintf("%s_%s", task._type.String(), task.delay), func(t *testing.T) {
			// Should not be able to run until delay is zero or negative
			require.False(t, task.setRunning())

			// Wait until delay is zero or negative
			require.Eventually(t, func() bool { return task.Delay() <= time.Duration(0) }, time.Second, time.Millisecond)

			// Move to Running
			require.True(t, task.setRunning())
			require.False(t, task.setRunning())
			require.False(t, task.Done())
			require.False(t, task.Canceled())

			// Get it ready for next execution
			task.configureNextRun()
			require.Greater(t, task.Delay(), time.Duration(0))
			require.False(t, task.Done())
			require.False(t, task.Canceled())
			require.NoError(t, task.Err())
			require.Nil(t, task.Value())
		})
	}

}

func TestTask_CancelBeforeRun(t *testing.T) {
	for _, task := range []*_Task{
		newTaskOnce(t, 0),
		newTaskOnce(t, time.Millisecond*10),
		newTaskFixedDelay(t, time.Millisecond*10),
	} {
		t.Run(fmt.Sprintf("%s_%s", task._type.String(), task.delay), func(t *testing.T) {
			// Cancel
			require.True(t, task.Cancel())
			require.False(t, task.Cancel())
			require.True(t, task.Canceled())

			// Check it is Done
			require.Eventually(t, func() bool { <-task.Await(); return true }, time.Second, 1)
			require.True(t, task.Done())
			require.ErrorIs(t, task.Err(), context.Canceled)
			require.Nil(t, task.Value())
		})
	}
}

func TestTask_CancelWhileRun(t *testing.T) {
	for _, task := range []*_Task{
		newTaskOnce(t, 0),
		newTaskOnce(t, time.Millisecond*10),
		newTaskFixedDelay(t, time.Millisecond*10),
	} {
		t.Run(fmt.Sprintf("%s_%s", task._type.String(), task.delay), func(t *testing.T) {

			// Should not be able to run until delay is zero or negative
			if task.Delay() > 0 {
				require.False(t, task.setRunning())

				// Wait until delay is zero or negative
				require.Eventually(t, func() bool { return task.Delay() <= time.Duration(0) }, time.Second, time.Millisecond)
			}

			// Move to Running
			require.True(t, task.setRunning())
			require.False(t, task.setRunning())
			require.False(t, task.Done())
			require.False(t, task.Canceled())

			// Cancel
			require.True(t, task.Cancel())
			require.False(t, task.Cancel())
			require.True(t, task.Canceled())
			require.False(t, task.Done())

			// Move to Done with error in go routine to simulate actual use
			time.AfterFunc(time.Millisecond, func() {
				task.setDone(nil, errors.Newf("TEST_ERROR", "test error"))
			})

			// Check it is Done
			require.Eventually(t, func() bool { <-task.Await(); return true }, time.Second, 1)
			require.True(t, task.Done())
			require.Error(t, task.Err())
			require.True(t, errors.IsCode(task.Err(), "TEST_ERROR"))
			require.Nil(t, task.Value())
		})
	}

}

func TestTask_Once_ConfigureNextRun(t *testing.T) {
	task := newTaskOnce(t, 0)
	require.PanicsWithValue(t, "Cannot configure next run for a task that is not periodic", func() { task.configureNextRun() })
}

func TestTask_Periodic_ConfigureNextRun(t *testing.T) {
	for _, task := range []*_Task{
		// FixedRate must go first in the list to avoid using a larger delay
		newTaskFixedRate(t, time.Millisecond*10),
		newTaskFixedDelay(t, time.Millisecond*10),
	} {
		t.Run(fmt.Sprintf("%s_%s", task._type.String(), task.delay), func(t *testing.T) {
			// Should panic trying to configure the next run if not running
			require.PanicsWithValue(t, "Cannot configure next run, current status: 0", func() { task.configureNextRun() })

			// Wait until delay is zero or negative
			require.Eventually(t, func() bool { return task.Delay() <= time.Duration(0) }, time.Second, time.Millisecond)

			// Move to Running
			require.True(t, task.setRunning())
			require.False(t, task.setRunning())

			// Should work fine now
			task.configureNextRun()
			require.Greater(t, task.Delay(), time.Duration(0))
			require.False(t, task.Done())

			// Wait until delay is zero or negative
			require.Eventually(t, func() bool { return task.Delay() <= time.Duration(0) }, time.Second, time.Millisecond)

			// Move To Running
			require.True(t, task.setRunning())
			require.False(t, task.setRunning())

			// Cancel
			require.True(t, task.Cancel())
			require.False(t, task.Cancel())
			require.True(t, task.Canceled())

			// Should panic trying to configure the next run if canceled
			require.PanicsWithValue(t,
				"Cannot configure next run for a task that is done or cancelling, current status: 2",
				func() { task.configureNextRun() },
			)
		})
	}
}

var seq = atomic.Uint64{}

func newTaskOnce(t *testing.T, delay time.Duration) *_Task {
	task := newTask(seq.Add(1), _TaskTypeOnce, delay, delay, make(chan _Event, 1))
	require.NotNil(t, task)
	require.False(t, task.Periodic())
	require.False(t, task.Done())
	require.False(t, task.Canceled())
	require.LessOrEqual(t, task.Delay(), delay)
	require.NoError(t, task.Err())
	require.Nil(t, task.Value())
	return task
}

func newTaskFixedDelay(t *testing.T, delay time.Duration) *_Task {
	task := newTask(seq.Add(1), _TaskTypeFixedDelay, delay, delay, make(chan _Event, 1))
	require.NotNil(t, task)
	require.True(t, task.Periodic())
	require.False(t, task.Done())
	require.False(t, task.Canceled())
	require.LessOrEqual(t, task.Delay(), delay)
	require.Greater(t, task.Delay(), time.Duration(0))
	require.NoError(t, task.Err())
	require.Nil(t, task.Value())
	return task
}

func newTaskFixedRate(t *testing.T, delay time.Duration) *_Task {
	task := newTask(seq.Add(1), _TaskTypeFixedRate, delay, delay, make(chan _Event, 1))
	require.NotNil(t, task)
	require.True(t, task.Periodic())
	require.False(t, task.Done())
	require.False(t, task.Canceled())
	require.LessOrEqual(t, task.Delay(), delay)
	require.Greater(t, task.Delay(), time.Duration(0))
	require.NoError(t, task.Err())
	require.Nil(t, task.Value())
	return task
}
