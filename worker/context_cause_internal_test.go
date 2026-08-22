package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/distributedlock"
	fwerrors "github.com/southernlabs-io/go-fw/errors"
)

type lockLossFactory struct {
	cause           error
	autoExtendCalls int
}

func (f *lockLossFactory) NewDistributedLock(resource string, ttl time.Duration) distributedlock.DistributedLock {
	return &lockLossLock{factory: f, resource: resource, ttl: ttl}
}

type lockLossLock struct {
	factory  *lockLossFactory
	resource string
	ttl      time.Duration
	cancel   context.CancelCauseFunc
}

func (l *lockLossLock) Resource() string                      { return l.resource }
func (l *lockLossLock) TTL() time.Duration                    { return l.ttl }
func (l *lockLossLock) Expiration() time.Time                 { return time.Now().Add(l.ttl) }
func (l *lockLossLock) ExtendedCount() int                    { return 0 }
func (l *lockLossLock) Lock(context.Context) error            { return nil }
func (l *lockLossLock) TryLock(context.Context) (bool, error) { return true, nil }
func (l *lockLossLock) Unlock(context.Context) error {
	l.cancel(context.Canceled)
	return nil
}
func (l *lockLossLock) Extend(context.Context) (bool, error) { return true, nil }
func (l *lockLossLock) AutoExtend(ctx context.Context) (context.Context, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	l.cancel = cancel
	l.factory.autoExtendCalls++
	if l.factory.autoExtendCalls == 1 {
		cancel(l.factory.cause)
	}
	return ctx, nil
}

type cancellationCauseWorker struct {
	runs     int
	finalErr error
}

func (w *cancellationCauseWorker) GetName() string { return "cancellation-cause-worker" }
func (w *cancellationCauseWorker) GetID() string   { return "test" }
func (w *cancellationCauseWorker) GetConcurrency() ConcurrencyConfig {
	return ConcurrencyConfig{Mode: ConcurrencyModeSingle, SingleLockTTL: time.Second}
}
func (w *cancellationCauseWorker) GetRetry() RetryConfig { return RetryConfig{} }
func (w *cancellationCauseWorker) Run(ctx context.Context) error {
	w.runs++
	if w.runs == 1 {
		<-ctx.Done()
		return ctx.Err()
	}
	return w.finalErr
}

func TestSingleWorkerRunnerPreservesLockLossCancellationCause(t *testing.T) {
	lockLossErr := fwerrors.Newf(distributedlock.ErrCodeLockNotAutoExtended, "lock lease expired")
	finalErr := errors.New("stop after lock reacquisition")
	factory := &lockLossFactory{cause: lockLossErr}
	probe := &cancellationCauseWorker{finalErr: finalErr}
	h := &LongRunningWorkerHandler{ctx: context.Background(), dlFactory: factory}

	err := h.singleWorkerRunner(NewWorkerContext(context.Background(), probe.GetName(), probe.GetID()), probe)

	require.ErrorIs(t, err, finalErr)
	require.Equal(t, 2, probe.runs)
}
