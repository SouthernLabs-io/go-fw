package sync_test

import (
	"context"
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
