package config_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	context2 "github.com/southernlabs-io/go-fw/context"
	"github.com/southernlabs-io/go-fw/errors"
)

func TestNoDeadlineContext(t *testing.T) {
	myErr := errors.Newf(errors.ErrCodeBadState, "8badfood")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(myErr)

	ndCtx := context2.NoDeadlineAndNotCancellableContext(ctx)
	require.NotNil(t, ndCtx)

	// Check it is not done
	require.Nil(t, ndCtx.Err())
	deadline, ok := ctx.Deadline()
	require.False(t, ok)
	require.Zero(t, deadline)

	// Cancel the parent context
	cancel(myErr)

	// Check the parent is done
	require.ErrorIs(t, ctx.Err(), context.Canceled)
	require.ErrorIs(t, context.Cause(ctx), myErr)

	// Check the no deadline context is not done nor have a cause
	require.NoError(t, ndCtx.Err())
	require.NoError(t, context.Cause(ndCtx))
	deadline, ok = ndCtx.Deadline()
	require.False(t, ok)
	require.Zero(t, deadline)
}
