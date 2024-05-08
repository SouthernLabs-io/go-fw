package worker_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/distributedlock"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/test"
	"github.com/southernlabs-io/go-fw/worker"
)

func TestLongRunningWorkerHandlerWithGoodAndBrokenWorker(t *testing.T) {
	var longRunningWorkerHandler *worker.LongRunningWorkerHandler
	app := test.FxUnit(
		t,
		distributedlock.ModuleLocal,
		worker.ModuleWorkerHandler,
		worker.ProvideAsLongRunningWorker(func() *TestLongRunningWorker {
			return NewTestLongRunningWorker("good worker")
		}),
		worker.ProvideAsLongRunningWorker(func() *TestLongRunningWorkerBroken {
			return NewTestLongRunningWorkerBroken("bad worker")
		}),
	).Populate(
		&longRunningWorkerHandler,
	)
	require.NotNil(t, longRunningWorkerHandler)

	sig := <-app.Wait()
	require.Equal(t, 1, sig.ExitCode)
}

func TestLongRunningWorkerHandlerWithBrokenWorker(t *testing.T) {
	var longRunningWorkerHandler *worker.LongRunningWorkerHandler
	app := test.FxUnit(
		t,
		distributedlock.ModuleLocal,
		worker.ModuleWorkerHandler,
		worker.ProvideAsLongRunningWorker(func() *TestLongRunningWorkerBroken {
			return NewTestLongRunningWorkerBroken("bad worker")
		}),
	).Populate(
		&longRunningWorkerHandler,
	)
	require.NotNil(t, longRunningWorkerHandler)

	sig := <-app.Wait()
	require.Equal(t, 1, sig.ExitCode)
}

func TestLongRunningWorkerHandlerStartStop(t *testing.T) {
	var target test.TargetBase
	var sd fx.Shutdowner
	var longRunningWorkerHandler *worker.LongRunningWorkerHandler
	fxApp := test.FxIntegration(
		t,
		distributedlock.ModulePostgres,
		worker.ProvideAsLongRunningWorker(func() *TestLongRunningWorker {
			return NewTestLongRunningWorker("good worker")
		}),
	).
		WithWorkerHandler().
		WithDB().
		Populate(&target, &sd, &longRunningWorkerHandler)
	require.NotNil(t, sd)
	require.NotNil(t, longRunningWorkerHandler)

	go func() {
		time.Sleep(5 * time.Second)
		err := sd.Shutdown()
		require.NoError(t, err)
	}()

	sig := <-fxApp.Wait()
	require.Equal(t, 0, sig.ExitCode)
}

func TestLongRunningWorkerNoWorker(t *testing.T) {
	var longRunningWorkerHandler *worker.LongRunningWorkerHandler
	app := test.FxUnit(
		t,
		distributedlock.ModuleLocal,
		worker.ModuleWorkerHandler,
	).Populate(
		&longRunningWorkerHandler,
	)
	require.NotNil(t, longRunningWorkerHandler)

	sig := <-app.Wait()
	require.Equal(t, 1, sig.ExitCode)
}

type TestLongRunningWorker struct {
	name              string
	id                string
	concurrencyConfig worker.ConcurrencyConfig
}

var _ worker.LongRunningWorker = TestLongRunningWorker{}

type TestLongRunningWorkerBroken struct {
	TestLongRunningWorker
}

func (t TestLongRunningWorker) GetName() string {
	return t.name
}

func (t TestLongRunningWorker) GetID() string {
	return t.id
}

func (t TestLongRunningWorker) GetConcurrency() worker.ConcurrencyConfig {
	return t.concurrencyConfig
}

func (t TestLongRunningWorker) Run(ctx context.Context) error {
	logger := log.GetLoggerFromCtx(ctx)
	for {
		logger.Debug("TestLongRunningWorker running")

		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		default:
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func NewTestLongRunningWorker(name string) *TestLongRunningWorker {
	return &TestLongRunningWorker{
		name: name,
		id:   uuid.NewString(),
		concurrencyConfig: worker.ConcurrencyConfig{
			Mode:          worker.ConcurrencyModeSingle,
			SingleLockTTL: time.Second * 10,
		},
	}
}

func (t TestLongRunningWorkerBroken) Run(ctx context.Context) error {
	log.GetLoggerFromCtx(ctx).Debug("quiting")
	return nil
}

func NewTestLongRunningWorkerBroken(name string) *TestLongRunningWorkerBroken {
	return &TestLongRunningWorkerBroken{
		TestLongRunningWorker: TestLongRunningWorker{
			name: name,
			id:   uuid.NewString(),
		},
	}
}
