package executors

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/queue"
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
	ctxCancel context.CancelFunc

	// executor events channel, it will be used to communicate back changes of state of the task
	eventChn chan _Event

	id      uint64
	_type   _TaskType
	status  atomic.Int32
	nextRun time.Time
	delay   time.Duration

	mu   sync.Mutex
	done atomic.Value

	canceled bool
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
	eventChn chan _Event,
) *_Task {
	return &_Task{
		id:    id,
		_type: taskType,

		nextRun: time.Now().Add(initialDelay), // First run is after the initial delay
		delay:   delay,

		eventChn: eventChn,

		queueElement:  nil,
		sQueueElement: nil,
	}
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
	return time.Until(t.nextRun)
}

func (t *_Task) Cancel() bool {
	if t.status.Load() >= _TaskStatusCancelling {
		return false
	}

	if t.status.CompareAndSwap(_TaskStatusNew, _TaskStatusDone) {
		t.err = context.Canceled
		t.canceled = true
		if t.ctxCancel != nil {
			t.ctxCancel()
		}
		doneChn := t.done.Load()
		if doneChn != nil && doneChn != closedChn {
			close(doneChn.(chan struct{}))
		}

		// Notify the executor
		t.eventChn <- _TaskCanceledEvent{Task: t}

		return true
	}

	if t.status.CompareAndSwap(_TaskStatusRunning, _TaskStatusCancelling) {
		t.canceled = true
		if t.ctxCancel != nil {
			t.ctxCancel()
		}
		// Notify the executor
		t.eventChn <- _TaskCanceledEvent{Task: t}

		return true
	}

	return false
}

func (t *_Task) Canceled() bool {
	return t.canceled
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

func (t *_Task) configureContext(parentCtx context.Context, name string) {
	t.ctx, t.ctxCancel = context.WithCancel(core.NewWorkerContext(
		parentCtx, name, fmt.Sprintf("%d@%s", t.id, core.CachedHostname()),
	))
}

func (t *_Task) configureNextRun() {
	if !t.Periodic() {
		panic("Cannot configure next run for a task that is not periodic")
	}

	if t.status.Load() >= _TaskStatusCancelling {
		panic(
			"Cannot configure next run for a task that is done or cancelling, current status: " +
				fmt.Sprintf("%d", t.status.Load()),
		)
	}

	if !t.status.CompareAndSwap(_TaskStatusRunning, _TaskStatusNew) {
		panic(
			"Cannot configure next run, current status: " +
				fmt.Sprintf("%d", t.status.Load()),
		)
	}
	switch t._type {
	case _TaskTypeFixedDelay:
		t.nextRun = time.Now().Add(t.delay)
	case _TaskTypeFixedRate:
		t.nextRun = t.nextRun.Add(t.delay)
	default:
		panic("Unknown task type: " + fmt.Sprintf("%d", t._type))
	}
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
	if t.ctxCancel != nil {
		t.ctxCancel()
	}
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
