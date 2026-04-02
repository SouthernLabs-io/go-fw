package log_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/log"
)

func TestCtxAppendLoggerAttrs_Empty(t *testing.T) {
	ctx := context.Background()
	ctx = log.CtxAppendLoggerAttrs(ctx, slog.String("key", "value"))
	attrs := log.GetLoggerAttrsFromCtx(ctx)
	require.Len(t, attrs, 1)
	require.Equal(t, "key", attrs[0].Key)
	require.Equal(t, "value", attrs[0].Value.String())
}

func TestCtxAppendLoggerAttrs_AppendDifferentKeys(t *testing.T) {
	ctx := context.Background()
	ctx = log.CtxAppendLoggerAttrs(ctx, slog.String("a", "1"))
	ctx = log.CtxAppendLoggerAttrs(ctx, slog.String("b", "2"))
	attrs := log.GetLoggerAttrsFromCtx(ctx)
	require.Len(t, attrs, 2)
	require.Equal(t, "a", attrs[0].Key)
	require.Equal(t, "b", attrs[1].Key)
}

func TestCtxAppendLoggerAttrs_DeduplicateSameKeyAndValue(t *testing.T) {
	ctx := context.Background()
	ctx = log.CtxAppendLoggerAttrs(ctx, slog.String("user.id", "abc"))
	ctx = log.CtxAppendLoggerAttrs(ctx, slog.String("user.id", "abc"))
	attrs := log.GetLoggerAttrsFromCtx(ctx)
	require.Len(t, attrs, 1)
	require.Equal(t, "user.id", attrs[0].Key)
	require.Equal(t, "abc", attrs[0].Value.String())
}

func TestCtxAppendLoggerAttrs_ReplacesDifferentValue(t *testing.T) {
	ctx := context.Background()
	ctx = log.CtxAppendLoggerAttrs(ctx, slog.String("project.id", "old"))
	ctx = log.CtxAppendLoggerAttrs(ctx, slog.String("project.id", "new"))
	attrs := log.GetLoggerAttrsFromCtx(ctx)
	require.Len(t, attrs, 1)
	require.Equal(t, "project.id", attrs[0].Key)
	require.Equal(t, "new", attrs[0].Value.String())
}

func TestCtxAppendLoggerAttrs_PreservesOrder(t *testing.T) {
	ctx := context.Background()
	ctx = log.CtxAppendLoggerAttrs(ctx,
		slog.String("a", "1"),
		slog.String("b", "2"),
		slog.String("c", "3"),
	)
	ctx = log.CtxAppendLoggerAttrs(ctx, slog.String("b", "updated"))
	attrs := log.GetLoggerAttrsFromCtx(ctx)
	require.Len(t, attrs, 3)
	require.Equal(t, "a", attrs[0].Key)
	require.Equal(t, "b", attrs[1].Key)
	require.Equal(t, "updated", attrs[1].Value.String())
	require.Equal(t, "c", attrs[2].Key)
}

func TestCtxAppendLoggerAttrs_GroupAttrsUntouched(t *testing.T) {
	ctx := context.Background()
	ctx = log.CtxAppendLoggerAttrs(ctx, slog.Group("usr", slog.String("id", "123")))
	ctx = log.CtxAppendLoggerAttrs(ctx, slog.String("user.id", "123"))
	attrs := log.GetLoggerAttrsFromCtx(ctx)
	require.Len(t, attrs, 2)
	require.Equal(t, "usr", attrs[0].Key)
	require.Equal(t, "user.id", attrs[1].Key)
}

func TestCtxAppendLoggerAttrs_MultipleNewAttrs(t *testing.T) {
	ctx := context.Background()
	ctx = log.CtxAppendLoggerAttrs(ctx, slog.String("a", "1"))
	ctx = log.CtxAppendLoggerAttrs(ctx,
		slog.String("a", "1"), // duplicate, skip
		slog.String("b", "2"), // new
		slog.String("a", "1"), // duplicate again within same call — first match wins
	)
	attrs := log.GetLoggerAttrsFromCtx(ctx)
	// "a" stays at index 0 (unchanged), "b" appended
	require.Len(t, attrs, 2)
	require.Equal(t, "a", attrs[0].Key)
	require.Equal(t, "1", attrs[0].Value.String())
	require.Equal(t, "b", attrs[1].Key)
}

func TestGetLoggerAttrsFromCtx_Empty(t *testing.T) {
	ctx := context.Background()
	attrs := log.GetLoggerAttrsFromCtx(ctx)
	require.Nil(t, attrs)
}

func TestCtxWithLoggerAttrs_Overwrites(t *testing.T) {
	ctx := context.Background()
	ctx = log.CtxAppendLoggerAttrs(ctx, slog.String("a", "1"), slog.String("b", "2"))
	ctx = log.CtxWithLoggerAttrs(ctx, slog.String("c", "3"))
	attrs := log.GetLoggerAttrsFromCtx(ctx)
	require.Len(t, attrs, 1)
	require.Equal(t, "c", attrs[0].Key)
}
