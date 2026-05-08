package sync_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/sync"
)

func TestSleep(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	err := sync.Sleep(ctx, time.Millisecond)
	require.NoError(t, err)
	require.True(t, time.Since(now) >= time.Millisecond)
}

func TestSleepCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := sync.Sleep(ctx, time.Millisecond)
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
}

func TestSleepCanceledWithCause(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	causeErr := errors.NewUnknownf("cause")
	cancel(causeErr)
	err := sync.Sleep(ctx, time.Millisecond)
	require.Error(t, err)
	require.ErrorIs(t, err, causeErr)
}

func TestOnceValue(t *testing.T) {
	count := &atomic.Int32{}
	onceValue := sync.OnceValue(func() string {
		count.Add(1)
		return "value"
	})

	require.Equal(t, "value", onceValue())
	require.Equal(t, "value", onceValue())
	require.Equal(t, int32(1), count.Load())
}

func TestOnceValues(t *testing.T) {
	count := &atomic.Int32{}
	onceValues := sync.OnceValues(func() (string, int) {
		count.Add(1)
		return "value", 42
	})

	value, number := onceValues()
	require.Equal(t, "value", value)
	require.Equal(t, 42, number)

	value, number = onceValues()
	require.Equal(t, "value", value)
	require.Equal(t, 42, number)
	require.Equal(t, int32(1), count.Load())
}
