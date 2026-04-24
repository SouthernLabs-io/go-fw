package context

import (
	"context"
	"time"
)

var ExtendableDeadlineCtxKey = CtxKey("_fw_extendable_deadline")

type extendableDeadline struct {
	cancelableCtx Context
	cancel        CancelCauseFunc
	timer         *time.Timer
	deadline      time.Time
}

func WithExtendableDeadline(parent Context, deadline time.Time) (Context, CancelFunc) {
	return WithExtendableTimeout(parent, time.Until(deadline))
}

func WithExtendableTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
	cancelableCtx, cancel := context.WithCancelCause(parent)
	timer := time.AfterFunc(timeout, func() {
		cancel(context.DeadlineExceeded)
	})

	return &extendableDeadline{
			deadline:      time.Now().Add(timeout),
			cancelableCtx: cancelableCtx,
			cancel:        cancel,
			timer:         timer,
		}, func() {
			if timer.Stop() {
				cancel(nil)
			}
		}
}

func ExtendDeadline(ctx context.Context, deadline time.Time) bool {
	eCtx, ok := ctx.(*extendableDeadline)
	if !ok {
		// Find as value
		eCtx, ok = ctx.Value(ExtendableDeadlineCtxKey).(*extendableDeadline)
		if !ok {
			return false
		}
	}
	if eCtx.timer == nil {
		return false
	}
	if eCtx.timer.Reset(time.Until(deadline)) {
		eCtx.deadline = deadline
		return true
	}

	return false
}

var _ context.Context = (*extendableDeadline)(nil)

func (e *extendableDeadline) Deadline() (deadline time.Time, ok bool) {
	return e.deadline, true
}

func (e *extendableDeadline) Done() <-chan struct{} {
	return e.cancelableCtx.Done()
}

// Value implements [context.Context].
func (e *extendableDeadline) Value(key any) any {
	if key == ExtendableDeadlineCtxKey {
		return e
	}
	return e.cancelableCtx.Value(key)
}

func (e *extendableDeadline) Err() error {
	return e.cancelableCtx.Err()
}
