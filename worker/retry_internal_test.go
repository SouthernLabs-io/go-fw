package worker

import (
	stdErrors "errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/context"
	"github.com/southernlabs-io/go-fw/distributedlock"
)

type retryDelayProbeWorker struct {
	name      string
	id        string
	mode      ConcurrencyMode
	retry     RetryConfig
	steps     []runStep
	stepIdx   int
	attempts  []time.Time
	attemptsM sync.Mutex
}

type runStep struct {
	runFor time.Duration
	err    error
}

var _ LongRunningWorker = (*retryDelayProbeWorker)(nil)

func (w *retryDelayProbeWorker) GetName() string {
	return w.name
}

func (w *retryDelayProbeWorker) GetID() string {
	return w.id
}

func (w *retryDelayProbeWorker) GetConcurrency() ConcurrencyConfig {
	mode := w.mode
	if mode != ConcurrencyModeMulti {
		mode = ConcurrencyModeSingle
	}

	return ConcurrencyConfig{
		Mode:          mode,
		SingleLockTTL: 200 * time.Millisecond,
	}
}

func (w *retryDelayProbeWorker) GetRetry() RetryConfig {
	return w.retry
}

func (w *retryDelayProbeWorker) Run(ctx context.Context) error {
	w.attemptsM.Lock()
	defer w.attemptsM.Unlock()

	w.attempts = append(w.attempts, time.Now())
	if len(w.steps) > 0 {
		idx := w.stepIdx
		if idx >= len(w.steps) {
			return stdErrors.New("fatal")
		}
		w.stepIdx++
		step := w.steps[idx]
		if step.runFor > 0 {
			time.Sleep(step.runFor)
		}
		return step.err
	}

	if len(w.attempts) <= 2 {
		// Transient string so the retry path is exercised via errors.IsTransient.
		return stdErrors.New("connection refused")
	}

	// Non-transient to stop the retry loop.
	return stdErrors.New("fatal")
}

func (w *retryDelayProbeWorker) Attempts() []time.Time {
	w.attemptsM.Lock()
	defer w.attemptsM.Unlock()

	copyAttempts := make([]time.Time, len(w.attempts))
	copy(copyAttempts, w.attempts)
	return copyAttempts
}

func TestSingleWorkerRunnerRetryConfigDelayBackoff(t *testing.T) {
	baseDelay := 25 * time.Millisecond
	probeWorker := &retryDelayProbeWorker{
		name: "retry-delay-probe",
		id:   uuid.NewString(),
		retry: RetryConfig{
			Delay:      baseDelay,
			MaxRetries: 2,
		},
	}

	h := &LongRunningWorkerHandler{
		ctx:       context.Background(),
		dlFactory: distributedlock.NewLocalFactory(),
	}

	err := h.singleWorkerRunner(NewWorkerContext(context.Background(), probeWorker.GetName(), probeWorker.GetID()), probeWorker)
	require.Error(t, err)

	attempts := probeWorker.Attempts()
	require.Len(t, attempts, 3)

	firstRetryDelay := attempts[1].Sub(attempts[0])
	secondRetryDelay := attempts[2].Sub(attempts[1])

	require.GreaterOrEqual(t, firstRetryDelay, baseDelay)
	require.GreaterOrEqual(t, secondRetryDelay, 2*baseDelay)

	// Generous upper bounds keep the assertion stable while still validating expected order of magnitude.
	require.Less(t, firstRetryDelay, 300*time.Millisecond)
	require.Less(t, secondRetryDelay, 350*time.Millisecond)
}

func TestSingleWorkerRunnerResetRetryCountDelay(t *testing.T) {
	baseDelay := 12 * time.Millisecond
	resetDelay := 20 * time.Millisecond
	transientErr := stdErrors.New("connection refused")

	probeWorker := &retryDelayProbeWorker{
		name: "retry-reset-delay-probe",
		id:   uuid.NewString(),
		retry: RetryConfig{
			Delay:                baseDelay,
			MaxRetries:           1,
			ResetRetryCountDelay: resetDelay,
		},
		steps: []runStep{
			{runFor: 0, err: transientErr},
			{runFor: 35 * time.Millisecond, err: transientErr},
			{runFor: 0, err: transientErr},
		},
	}

	h := &LongRunningWorkerHandler{
		ctx:       context.Background(),
		dlFactory: distributedlock.NewLocalFactory(),
	}

	err := h.singleWorkerRunner(NewWorkerContext(context.Background(), probeWorker.GetName(), probeWorker.GetID()), probeWorker)
	require.Error(t, err)

	attempts := probeWorker.Attempts()
	// If reset is not applied after the long second run, max retries=1 would stop at 2 attempts.
	require.Len(t, attempts, 3)

	firstRetryDelay := attempts[1].Sub(attempts[0])
	require.GreaterOrEqual(t, firstRetryDelay, baseDelay)
	require.Less(t, firstRetryDelay, 250*time.Millisecond)
}

func TestMultiWorkerRunnerRetryConfigDelayBackoff(t *testing.T) {
	baseDelay := 20 * time.Millisecond
	probeWorker := &retryDelayProbeWorker{
		name: "multi-retry-delay-probe",
		id:   uuid.NewString(),
		mode: ConcurrencyModeMulti,
		retry: RetryConfig{
			Delay:      baseDelay,
			MaxRetries: 2,
		},
	}

	h := &LongRunningWorkerHandler{ctx: context.Background()}

	err := h.multiWorkerRunner(NewWorkerContext(context.Background(), probeWorker.GetName(), probeWorker.GetID()), probeWorker)
	require.Error(t, err)

	attempts := probeWorker.Attempts()
	require.Len(t, attempts, 3)

	firstRetryDelay := attempts[1].Sub(attempts[0])
	secondRetryDelay := attempts[2].Sub(attempts[1])

	require.GreaterOrEqual(t, firstRetryDelay, baseDelay)
	require.GreaterOrEqual(t, secondRetryDelay, 2*baseDelay)

	require.Less(t, firstRetryDelay, 300*time.Millisecond)
	require.Less(t, secondRetryDelay, 350*time.Millisecond)
}
