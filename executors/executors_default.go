package executors

import (
	"container/list"
	"context"
	"math"
	"sync/atomic"
	"time"

	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/queue"
)

// The transitions are expected to go sequentially from Running -> ShuttingDown -> WaitingTermination -> Terminated
const (
	_ExecutorStatusRunning = iota
	_ExecutorStatusShuttingDown
	_ExecutorStatusWaitingTermination
	_ExecutorStatusTerminated
)

type _Event interface {
	private()
}

type _PrivateEvent struct {
}

func (e _PrivateEvent) private() {
}

type _TaskCompletedEvent struct {
	_PrivateEvent
	Task *_Task
}

type _TaskCanceledEvent struct {
	_PrivateEvent
	Task *_Task
}

type _TaskSubmittedEvent struct {
	_PrivateEvent
	Task *_Task
}

type _CancelEvent struct {
	_PrivateEvent
	Wait chan bool
}

type _UpdateConcurrencyEvent struct {
	_PrivateEvent
	Concurrency int
	Wait        chan struct{}
}

type _UpdateQueueCapacityEvent struct {
	_PrivateEvent
	Capacity int
	Wait     chan struct{}
}

type _CancelNowEvent struct {
	_PrivateEvent
	Wait chan []Future
}

type DefaultExecutor struct {
	ctx    context.Context
	cancel context.CancelFunc

	status atomic.Int32

	queue                *list.List
	scheduledQueue       *queue.PriorityQueue[*_Task]
	schedulerTimer       *time.Timer
	schedulerTimerTarget time.Time

	queueCapacity int
	concurrency   int
	running       int
	taskCount     atomic.Uint64

	terminationChn chan struct{}
	eventChn       chan _Event
}

var _ Executor = (*DefaultExecutor)(nil)

/*
NewDefaultExecutor creates a new executor with the given concurrency and queue capacity. Either number can be negative which
means there is no limit on the number of concurrent tasks or the queue capacity.

The executor will run until the given context is canceled or the Executor.Cancel/Executor.CancelNow methods are called.

When the capacity of the queue is reached, the executor will reject new tasks by returning ErrExecutorQueueFull.

When the executor is canceled, it will reject new tasks by returning ErrExecutorCanceled.

Implementation notes:
- The executor uses a priority queue to schedule tasks based on how soon they need to be executed.
- The priority queue is implemented using a heap.
- The executor uses a timer to schedule the next task to run by inspecting the Top Element from the priority queue.
- The executor uses a linked list to store tasks that are ready to run but there is no room to execute them.
- Tasks are removed from the queue when they are started or canceled.
*/
func NewDefaultExecutor(ctx context.Context, concurrency int, queueCapacity int) *DefaultExecutor {
	ctx, cancel := context.WithCancel(ctx)
	e := &DefaultExecutor{
		ctx:    ctx,
		cancel: cancel,

		queue:                list.New(),
		scheduledQueue:       queue.NewPriorityQueue[*_Task](),
		schedulerTimer:       time.NewTimer(0),
		schedulerTimerTarget: time.Now(),

		concurrency:   concurrency,
		queueCapacity: queueCapacity,

		terminationChn: make(chan struct{}),
		eventChn:       make(chan _Event),
	}

	e.status.Store(_ExecutorStatusRunning)

	// Start the internal event loop
	go e.eventLoop()

	return e
}

func (e *DefaultExecutor) Concurrency() int {
	return e.concurrency
}

func (e *DefaultExecutor) SetConcurrency(concurrency int) {
	event := _UpdateConcurrencyEvent{Concurrency: concurrency, Wait: make(chan struct{})}
	e.eventChn <- event
	<-event.Wait
}

func (e *DefaultExecutor) QueueLength() int {
	return e.queue.Len() + e.scheduledQueue.Len()
}

func (e *DefaultExecutor) QueueCapacity() int {
	return e.queueCapacity
}

func (e *DefaultExecutor) SetQueueCapacity(capacity int) {
	event := _UpdateQueueCapacityEvent{Capacity: capacity, Wait: make(chan struct{})}
	e.eventChn <- event
	<-event.Wait
}

func (e *DefaultExecutor) Submit(runnable Runnable) (Future, error) {
	return e.Schedule(runnable, 0)
}

func (e *DefaultExecutor) SubmitProducer(callable Producer) (ProducerFuture, error) {
	return e.schedule(
		callable,
		0,
		0,
		_TaskTypeOnce,
	)
}

func (e *DefaultExecutor) Schedule(runnable Runnable, delay time.Duration) (ScheduledFuture, error) {
	return e.schedule(
		func(ctx context.Context) (any, error) { return nil, runnable(ctx) },
		delay,
		0,
		_TaskTypeOnce,
	)
}

func (e *DefaultExecutor) ScheduleWithFixedRate(
	runnable Runnable,
	initialDelay time.Duration,
	period time.Duration,
) (ScheduledFuture, error) {
	return e.schedule(
		func(ctx context.Context) (any, error) { return nil, runnable(ctx) },
		initialDelay,
		period,
		_TaskTypeFixedRate,
	)
}

func (e *DefaultExecutor) ScheduleWithFixedDelay(
	runnable Runnable,
	initialDelay time.Duration,
	delay time.Duration,
) (ScheduledFuture, error) {
	return e.schedule(
		func(ctx context.Context) (any, error) { return nil, runnable(ctx) },
		initialDelay,
		delay,
		_TaskTypeFixedDelay,
	)
}

func (e *DefaultExecutor) schedule(
	callable Producer,
	initialDelay time.Duration,
	delay time.Duration,
	taskType _TaskType,
) (*_Task, error) {
	if e.Canceled() {
		return nil, ErrExecutorCanceled
	}

	// Best effort to keep the queue size under the capacity. Two concurrent goroutines adding tasks to the queue
	// can cause the queue to grow beyond the capacity.
	if e.remainingTotalCapacity() < 1 {
		return nil, ErrExecutorQueueFull
	}

	taskID := e.taskCount.Add(1)
	task := newTask(taskID, taskType, initialDelay, delay, e.eventChn)
	task.execute = callable
	task.configureContext(e.ctx, "default-executor-task")
	e.eventChn <- _TaskSubmittedEvent{Task: task}
	return task, nil
}

func (e *DefaultExecutor) Terminated() bool {
	return e.status.Load() >= _ExecutorStatusTerminated
}

func (e *DefaultExecutor) Canceled() bool {
	return e.status.Load() >= _ExecutorStatusShuttingDown
}

func (e *DefaultExecutor) Cancel() bool {
	if e.status.Load() >= _ExecutorStatusShuttingDown {
		return false
	}
	event := _CancelEvent{Wait: make(chan bool)}
	e.eventChn <- event
	return <-event.Wait
}

func (e *DefaultExecutor) CancelNow() []Future {
	if e.status.Load() >= _ExecutorStatusWaitingTermination {
		return nil
	}

	e.Cancel()

	event := _CancelNowEvent{
		Wait: make(chan []Future),
	}
	e.eventChn <- event
	futures := <-event.Wait
	return futures
}

func (e *DefaultExecutor) AwaitTermination(timeout time.Duration) bool {
	if e.status.Load() == _ExecutorStatusTerminated {
		return true
	}

	select {
	case <-e.terminationChn:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (e *DefaultExecutor) eventLoop() {
	for {
		select {

		// Check if the executor context was canceled
		case <-e.ctx.Done():
			// Must be called inside a goroutine to avoid locking the event loop
			// NOTE: we could use a buffered channel to avoid this, but then it is not clear what would be the buffer size
			go func() {
				e.Cancel()
			}()

		// Check if the scheduler timer fired
		case <-e.schedulerTimer.C:
			if e.scheduledQueue.Len() == 0 {
				continue
			}

			// Check if the next task is meant to be run now
			if e.scheduledQueue.Peek().Value.nextRun.Compare(time.Now()) <= 0 {
				task := e.scheduledQueue.Pop()
				task.Value.sQueueElement = nil
				e.tryExecuteTask(task.Value)
			}
			e.configureSchedulerTimer()

		// Check if there are events to process
		case eventAny := <-e.eventChn:
			switch event := eventAny.(type) {

			case _TaskSubmittedEvent:
				e.tryExecuteTask(event.Task)

			case _TaskCompletedEvent:
				e.running--
				task := event.Task
				if !task.Done() {
					if task.Canceled() {
						task.setDone(nil, context.Canceled)
					} else if task.Periodic() {
						task.configureNextRun()
						task.configureContext(e.ctx, "default-executor-task")
						if task.Delay() > 0 {
							e.scheduledQueue.Push(task, task.nextRun.UnixNano())
							e.configureSchedulerTimer()
						} else {
							e.queue.PushBack(task)
						}
					}
				}
				if !e.tryRunNextTask() {
					return
				}

			case _TaskCanceledEvent:
				task := event.Task
				if task.err == nil {
					task.err = context.Canceled
				}
				if e.queue.Len() == 0 {
					continue
				}

				// Remove canceled tasks from the queue
				if task.queueElement != nil {
					e.queue.Remove(task.queueElement)
				}
				// Remove from priority queue
				if task.sQueueElement != nil {
					e.scheduledQueue.Remove(task.sQueueElement)
				}

			case _UpdateConcurrencyEvent:
				e.concurrency = event.Concurrency
				close(event.Wait)
				for e.canStartNewTask() && e.queue.Len() > 0 {
					if !e.tryRunNextTask() {
						return
					}
				}

			case _UpdateQueueCapacityEvent:
				e.queueCapacity = event.Capacity
				close(event.Wait)

			case _CancelEvent:
				if e.status.Load() >= _ExecutorStatusShuttingDown {
					event.Wait <- false
					close(event.Wait)
					continue
				}
				event.Wait <- e.status.CompareAndSwap(_ExecutorStatusRunning, _ExecutorStatusShuttingDown)
				close(event.Wait)
				if !e.tryRunNextTask() {
					return
				}

			case _CancelNowEvent:
				if e.status.Load() >= _ExecutorStatusWaitingTermination {
					// Signal we are done
					close(event.Wait)
					continue
				}

				// Collect all tasks from the queue, run over them then reset the queue to avoid extra work on each remove
				var futures []Future
				for item := e.queue.Front(); item != nil; item = item.Next() {
					futures = append(futures, item.Value.(*_Task))
				}
				e.queue.Init()

				// Collect all tasks from the scheduled tasks
				for e.scheduledQueue.Len() > 0 {
					futures = append(futures, e.scheduledQueue.Pop().Value)
				}

				// Update status
				e.status.CompareAndSwap(_ExecutorStatusShuttingDown, _ExecutorStatusWaitingTermination)

				// Signal we are done
				event.Wait <- futures
				close(event.Wait)
				if !e.tryRunNextTask() {
					return
				}
			}
		}
	}
}

func (e *DefaultExecutor) tryRunNextTask() (shouldContinue bool) {
	if e.Terminated() {
		return true
	}

	if e.QueueLength() == 0 {
		if e.Canceled() {
			if e.running > 0 {
				e.status.CompareAndSwap(_ExecutorStatusShuttingDown, _ExecutorStatusWaitingTermination)
				return true
			} else {
				status := e.status.Load()
				if (status == _ExecutorStatusShuttingDown || status == _ExecutorStatusWaitingTermination) &&
					e.status.CompareAndSwap(status, _ExecutorStatusTerminated) {
					close(e.terminationChn)
					return false
				}
			}
		}
		return true
	} else if e.queue.Len() == 0 {
		// No tasks ready to run
		return true
	}

	// No more room to start new tasks
	if !e.canStartNewTask() {
		return true
	}

	// Run the next task
	task := e.queue.Remove(e.queue.Front()).(*_Task)
	task.queueElement = nil
	e.tryExecuteTask(task)
	return true
}

func (e *DefaultExecutor) tryExecuteTask(task *_Task) {
	// Check if this task can be run
	if !task.Runnable() {
		return
	}

	// Schedule if the task is meant to be run in the future
	if task.Delay() > 0 {
		task.sQueueElement = e.scheduledQueue.Push(task, task.nextRun.UnixNano())
		e.configureSchedulerTimer()
		return
	}

	if !e.canStartNewTask() {
		// No room to start new tasks, append to the queue
		task.queueElement = e.queue.PushBack(task)
	} else {
		e.executeTask(task)
	}
}

// RemainingTotalCapacity returns the number of new tasks that can be run concurrently and accepted in the queue
func (e *DefaultExecutor) remainingTotalCapacity() int {
	if e.concurrency < 0 || e.queueCapacity < 0 {
		return math.MaxInt32
	}
	return e.concurrency - e.running + e.queueCapacity - e.QueueLength()
}

// return true if we can start a new task
func (e *DefaultExecutor) canStartNewTask() bool {
	if e.concurrency < 0 {
		return true
	}
	return e.running < e.concurrency
}

func (e *DefaultExecutor) executeTask(task *_Task) {
	if !task.setRunning() {
		return
	}
	e.running++
	go func() {
		defer func() { e.eventChn <- _TaskCompletedEvent{Task: task} }()
		// handle panic
		defer func() {
			if r := recover(); r != nil {
				task.setDone(nil, errors.Newf(ErrCodeTaskPanic, "task failed with panic: %v", r))
			}
		}()

		res, err := task.execute(task.ctx)
		if err != nil || !task.Periodic() {
			task.setDone(res, err)
		}
	}()
}

func (e *DefaultExecutor) configureSchedulerTimer() {
	if e.scheduledQueue.Len() == 0 {
		return
	}
	nextRun := e.scheduledQueue.Peek().Value.nextRun
	if nextRun.Equal(e.schedulerTimerTarget) {
		return
	}

	// Following timer documentation on how properly reset a timer
	if !e.schedulerTimer.Stop() && len(e.schedulerTimer.C) > 0 {
		<-e.schedulerTimer.C
	}
	until := time.Until(nextRun)
	e.schedulerTimerTarget = nextRun
	e.schedulerTimer.Reset(until)
}
