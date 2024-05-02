package executors

import (
	"context"
	"time"

	"github.com/southernlabs-io/go-fw/errors"
)

/*
Future represents the result of an asynchronous computation.

Example:

	future, err := executor.Submit(func(ctx context.Context) error {
		// Do something
		return nil
	})
	if err != nil {
		// Handle error
	}
	// Do something else

	// Wait for the computation to complete
	<-future.Await()

	// Check if the computation was successful
	if future.Err() != nil {
		// Handle error
	}
*/
type Future interface {
	// Cancel will return true if the future execution was Canceled and Err will return context.Canceled.
	// A Future can only be Canceled if it is not Done. New calls to Cancel will always return false.
	Cancel() bool

	// Canceled returns true if the future was canceled.
	Canceled() bool

	// Await returns a channel that should be used to block until the computation is Done. It is safe to call Await
	// multiple times, it will always return the same channel.
	Await() <-chan struct{}

	// Done returns true if the computation is done, false otherwise.
	Done() bool

	// Err returns the error that caused the Future to be Done, or nil if it completed successfully.
	// An error with Code ErrCodeTaskPanic will be returned if the task panicked.
	Err() error
}

/*
ProducerFuture represents the result of an asynchronous computation that produces a Value.
*/
type ProducerFuture interface {
	Future

	// Value returns the computed value, it should only be called after the computation is done according to Future.IsDone
	// and there is no Error returned by Future.Err.
	Value() any
}

/*
ScheduledFuture represents the result of an asynchronous computation that was scheduled to run after a given delay.
Delay returns how much time is left before the computation is ready to execute. A zero or negative delay means it is
ready to execute.

Periodic returns true if the computation is scheduled to run periodically, false if it is scheduled to run only once.

A ScheduledFuture will not be Done until it is Canceled.
*/
type ScheduledFuture interface {
	Future
	Delay() time.Duration
	Periodic() bool
}

type Runnable func(ctx context.Context) error

type Producer func(ctx context.Context) (any, error)

var ErrExecutorQueueFull = errors.Newf("EXECUTOR_QUEUE_FULL", "executor queue is full")
var ErrExecutorCanceled = errors.Newf("EXECUTOR_CANCELED", "executor is canceled")

const ErrCodeTaskPanic = "TASK_PANIC"

/*
Executor can schedule commands to run after a given delay, or to execute periodically.
Executor also provides methods to manage its lifecycle.
*/
type Executor interface {

	// Concurrency returns the number of commands that can run concurrently.
	Concurrency() int

	// QueueLength returns the number of commands in the queue, it does not include running commands.
	QueueLength() int

	// Submit runs the command as soon as possible.
	// The command can be rejected with ErrExecutorQueueFull or ErrExecutorCanceled.
	Submit(runnable Runnable) (Future, error)

	// SubmitProducer runs the command as soon as possible. The produced value can be retrieved by calling Future.Value().
	// The command can be rejected with ErrExecutorQueueFull or ErrExecutorCanceled.
	SubmitProducer(callable Producer) (ProducerFuture, error)

	// Schedule runs the command after the given delay.
	// The command can be rejected with ErrExecutorQueueFull or ErrExecutorCanceled.
	Schedule(runnable Runnable, delay time.Duration) (ScheduledFuture, error)

	// ScheduleWithFixedDelay runs the command first after the given initial delay, and then repeatedly with the given
	// delay between the termination of one execution and the commencement of the next.
	// The command can be rejected with ErrExecutorQueueFull or ErrExecutorCanceled.
	ScheduleWithFixedDelay(runnable Runnable, initialDelay time.Duration, delay time.Duration) (ScheduledFuture, error)

	// ScheduleWithFixedRate runs the command first after the given initial delay, and then repeatedly with the given
	// period between the commencement of subsequent executions. A slow command will affect the start time of the next
	// execution if it takes longer than its period, in which case the next execution will start immediately after the
	// slow command finishes. Executions are always sequential, so no two executions will be running at the same time.
	// The command can be rejected with ErrExecutorQueueFull or ErrExecutorCanceled.
	ScheduleWithFixedRate(runnable Runnable, initialDelay time.Duration, period time.Duration) (ScheduledFuture, error)

	// Cancel initiates an orderly shutdown in which previously submitted commands are executed and new commands will
	// be rejected with ErrExecutorCanceled. This method does not wait for previously submitted commands to complete, see
	// AwaitTermination for that. It will return true if the executor successfully started the shutdown process.
	// Successive calls to Cancel do not have any effect and will always return false.
	Cancel() bool

	// Canceled returns true if the executor has been canceled.
	Canceled() bool

	// CancelNow initiates a shutdown as described in Cancel, but it will also cancel all running commands and return
	// all the enqueued commands. Cancellation is done by calling Future.Cancel() on each running command, this will be
	// effective only if the command is checking for cancellation by reading their context.Done() channel.
	// This method does not wait for running commands to complete, see AwaitTermination for that.
	CancelNow() []Future

	// AwaitTermination blocks until the queue is empty and there are no more running commands, or until the given timeout.
	// It returns true on successful termination, false if the timeout was reached.
	AwaitTermination(timeout time.Duration) bool

	// Terminated returns true if the executor has been canceled and the queue is empty and all commands have completed.
	Terminated() bool
}
