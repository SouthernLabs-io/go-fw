package context_test

import (
	stdcontext "context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/context"
)

func TestWithExtendableTimeout_DeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithExtendableTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	deadline, ok := ctx.Deadline()
	require.True(t, ok)
	require.WithinDuration(t, time.Now().Add(25*time.Millisecond), deadline, 30*time.Millisecond)

	require.Eventually(t, func() bool {
		return ctx.Err() == stdcontext.Canceled
	}, 300*time.Millisecond, time.Millisecond)
	require.ErrorIs(t, context.Cause(ctx), stdcontext.DeadlineExceeded)
}

func TestExtendDeadline_ExtendsCurrentDeadline(t *testing.T) {
	ctx, cancel := context.WithExtendableTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	extended := context.ExtendDeadline(ctx, time.Now().Add(90*time.Millisecond))
	require.True(t, extended)

	time.Sleep(40 * time.Millisecond)
	require.NoError(t, ctx.Err())

	require.Eventually(t, func() bool {
		return ctx.Err() == stdcontext.Canceled
	}, 250*time.Millisecond, time.Millisecond)
	require.ErrorIs(t, context.Cause(ctx), stdcontext.DeadlineExceeded)
}

func TestExtendDeadline_UsesValueLookupOnWrappedContext(t *testing.T) {
	base, cancel := context.WithExtendableTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	wrapped := stdcontext.WithValue(base, "key", "value")

	extended := context.ExtendDeadline(wrapped, time.Now().Add(80*time.Millisecond))
	require.True(t, extended)

	time.Sleep(35 * time.Millisecond)
	require.NoError(t, wrapped.Err())

	require.Eventually(t, func() bool {
		return wrapped.Err() == stdcontext.Canceled
	}, 250*time.Millisecond, time.Millisecond)
	require.ErrorIs(t, context.Cause(wrapped), stdcontext.DeadlineExceeded)
}

func TestExtendDeadline_ReturnsFalseForNonExtendableContext(t *testing.T) {
	ctx := stdcontext.Background()
	require.False(t, context.ExtendDeadline(ctx, time.Now().Add(time.Second)))
}

func TestWithExtendableTimeout_ManualCancelBeforeDeadline(t *testing.T) {
	ctx, cancel := context.WithExtendableTimeout(context.Background(), 200*time.Millisecond)

	cancel()

	require.Eventually(t, func() bool {
		return ctx.Err() == stdcontext.Canceled
	}, 100*time.Millisecond, time.Millisecond)
	require.ErrorIs(t, context.Cause(ctx), stdcontext.Canceled)
}

func TestWithExtendableDeadline_ManualCancel(t *testing.T) {
	ctx, cancel := context.WithExtendableDeadline(context.Background(), time.Now().Add(200*time.Millisecond))

	cancel()

	require.Eventually(t, func() bool {
		return ctx.Err() == stdcontext.Canceled
	}, 100*time.Millisecond, time.Millisecond)
	require.ErrorIs(t, context.Cause(ctx), stdcontext.Canceled)
}
