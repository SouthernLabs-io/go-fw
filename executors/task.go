package executors

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/queue"
	"github.com/southernlabs-io/go-fw/worker"
)

/*
The transitions are expected to happen in the following order:
  - New -> Running -> Done
  - New -> Done

A Done task can't be canceled.
*/
const (
	_TaskStatusNew = iota
	_TaskStatusRunning
	_TaskStatusCancelling
	_TaskStatusDone
)

type _TaskType int

const (
	_TaskTypeOnce _TaskType = iota
	_TaskTypeFixedDelay
	_TaskTypeFixedRate
)

// Reusable closed channel
var closedChn = make(chan struct{})

func init() {
	close(closedChn)
}

type _Task struct {
	ctx       context.Context
	ctxCancel context.CancelFunc // guarded by mu: Cancel calls it from any goroutine while the event loop re-arms periodic tasks

	// executor event sink used to communicate task state changes
	eventEmitter func(_Event) bool

	id      uint64
	_type   _TaskType
	status  atomic.Int32
	nextRun atomic.Int64 // unix nanos; the event loop re-arms it while Delay() reads it from callers
	delay   time.Duration

	mu   sync.Mutex
	done atomic.Value

	canceled atomic.Bool
	err      error
	value    any

	execute Producer

	queueElement  *list.Element
	sQueueElement *queue.Element[*_Task]
}

var _ Future = (*_Task)(nil)
var _ ScheduledFuture = (*_Task)(nil)

func newTask(
	id uint64,
	taskType _TaskType,
	initialDelay time.Duration,
	delay time.Duration,
	eventEmitter func(_Event) bool,
) *_Task {
	t := &_Task{
		id:    id,
		_type: taskType,

		delay: delay,

		eventEmitter: eventEmitter,

		queueElement:  nil,
		sQueueElement: nil,
	}
	t.nextRun.Store(time.Now().Add(initialDelay).UnixNano()) // First run is after the initial delay
	return t
}

func (t *_Task) Runnable() bool {
	return t.status.Load() == _TaskStatusNew
}

func (t *_Task) setRunning() bool {
	if !t.Runnable() || t.Delay() > time.Duration(0) {
		return false
	}

	return t.status.CompareAndSwap(_TaskStatusNew, _TaskStatusRunning)
}

func (t *_Task) Periodic() bool {
	return t._type == _TaskTypeFixedRate || t._type == _TaskTypeFixedDelay
}

func (t *_Task) Delay() time.Duration {
	return time.Until(t.nextRunTime())
}

func (t *_Task) nextRunTime() time.Time {
	return time.Unix(0, t.nextRun.Load())
}

func (t *_Task) cancelContext() {
	t.mu.Lock()
	cancel := t.ctxCancel
	t.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (t *_Task) Cancel() bool {
	if t.status.Load() >= _TaskStatusCancelling {
		return false
	}

	if t.status.CompareAndSwap(_TaskStatusNew, _TaskStatusDone) {
		t.err = context.Canceled
		t.canceled.Store(true)
		t.cancelContext()
		doneChn := t.done.Load()
		if doneChn != nil && doneChn != closedChn {
			close(doneChn.(chan struct{}))
		}

		// Best effort notification; executor may already be terminated.
		if t.eventEmitter != nil {
			t.eventEmitter(_TaskCanceledEvent{Task: t})
		}

		return true
	}

	if t.status.CompareAndSwap(_TaskStatusRunning, _TaskStatusCancelling) {
		t.canceled.Store(true)
		t.cancelContext()
		// Best effort notification; executor may already be terminated.
		if t.eventEmitter != nil {
			t.eventEmitter(_TaskCanceledEvent{Task: t})
		}

		return true
	}

	return false
}

func (t *_Task) Canceled() bool {
	return t.canceled.Load()
}

func (t *_Task) Await() <-chan struct{} {
	doneChn := t.done.Load()
	if doneChn != nil {
		return doneChn.(chan struct{})
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	doneChn = t.done.Load()
	if doneChn == nil {
		if t.status.Load() == _TaskStatusDone {
			doneChn = closedChn
		} else {
			doneChn = make(chan struct{})
		}
		t.done.Store(doneChn)
	}

	return doneChn.(chan struct{})
}

func (t *_Task) Done() bool {
	return t.status.Load() == _TaskStatusDone
}

func (t *_Task) Err() error {
	return t.err
}

func (t *_Task) Value() any {
	return t.value
}

func (t *_Task) configureContext(parentCtx context.Context, rootConf config.RootConfig, name string) {
	ctx, cancel := context.WithCancel(worker.NewWorkerContext(
		parentCtx, name, fmt.Sprintf("%d@%s", t.id, rootConf.Env.Host),
	))
	t.mu.Lock()
	t.ctx, t.ctxCancel = ctx, cancel
	t.mu.Unlock()
}

// configureNextRun re-arms a periodic task after a run; it returns false when the task was canceled
// while that run was finishing (Cancel moved it from Running to Cancelling), so it must not run again
func (t *_Task) configureNextRun() bool {
	if !t.Periodic() {
		panic("Cannot configure next run for a task that is not periodic")
	}

	if !t.status.CompareAndSwap(_TaskStatusRunning, _TaskStatusNew) {
		status := t.status.Load()
		if status == _TaskStatusCancelling {
			return false
		}
		panic("Cannot configure next run, current status: " + fmt.Sprintf("%d", status))
	}
	switch t._type {
	case _TaskTypeFixedDelay:
		t.nextRun.Store(time.Now().Add(t.delay).UnixNano())
	case _TaskTypeFixedRate:
		t.nextRun.Store(t.nextRunTime().Add(t.delay).UnixNano())
	default:
		panic("Unknown task type: " + fmt.Sprintf("%d", t._type))
	}
	return true
}

func (t *_Task) setDone(value any, err error) bool {
	status := t.status.Load()
	if status == _TaskStatusDone {
		return false
	}

	if !t.status.CompareAndSwap(status, _TaskStatusDone) {
		return false
	}

	t.value = value
	t.err = err
	t.cancelContext()
	doneChn := t.done.Load()
	if doneChn != nil {
		// quick path
		if doneChn != closedChn {
			close(doneChn.(chan struct{}))
		}
	} else if !t.done.CompareAndSwap(nil, closedChn) {
		// slow path
		t.mu.Lock()
		defer t.mu.Unlock()
		doneChn = t.done.Load()
		if doneChn != nil && doneChn != closedChn {
			close(doneChn.(chan struct{}))
		}
	}
	return true
}
