package databasebun_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	databasebun "github.com/southernlabs-io/go-fw/database/bun"
	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/log"
)

func newTestCtx(t *testing.T, buf *bytes.Buffer) context.Context {
	t.Helper()
	conf := config.RootConfig{Log: config.LogConfig{Level: config.LogLevelDebug, Structured: true}}
	lf := log.NewLoggerFactoryWithWriter(conf, buf)
	return lf.AddToCtx(context.Background())
}

func logLevel(t *testing.T, buf *bytes.Buffer) string {
	t.Helper()
	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	level, ok := entry["level"].(string)
	require.True(t, ok, "missing level field in log entry: %s", buf.String())
	return level
}

func queryEvent(err error) *bun.QueryEvent {
	return &bun.QueryEvent{
		DB:        (*bun.DB)(nil),
		Query:     "SELECT 1",
		StartTime: time.Now(),
		Err:       err,
	}
}

func TestBunLogger_AfterQuery_ContextCanceled_LogsDebug(t *testing.T) {
	buf := &bytes.Buffer{}
	ctx := newTestCtx(t, buf)
	logger := &databasebun.BunLogger{}

	logger.AfterQuery(ctx, queryEvent(context.Canceled))

	require.NotEmpty(t, buf.Bytes(), "expected a log entry")
	require.Equal(t, "DEBUG", logLevel(t, buf))
}

func TestBunLogger_AfterQuery_ContextCanceled_WrappedError_LogsDebug(t *testing.T) {
	buf := &bytes.Buffer{}
	ctx := newTestCtx(t, buf)
	logger := &databasebun.BunLogger{}

	wrapped := fmt.Errorf("query failed: %w", context.Canceled)
	logger.AfterQuery(ctx, queryEvent(wrapped))

	require.NotEmpty(t, buf.Bytes(), "expected a log entry")
	require.Equal(t, "DEBUG", logLevel(t, buf))
}

func TestBunLogger_AfterQuery_RegularError_LogsError(t *testing.T) {
	buf := &bytes.Buffer{}
	ctx := newTestCtx(t, buf)
	logger := &databasebun.BunLogger{}

	logger.AfterQuery(ctx, queryEvent(fmt.Errorf("connection refused")))

	require.NotEmpty(t, buf.Bytes(), "expected a log entry")
	require.Equal(t, "ERROR", logLevel(t, buf))
}

func TestBunLogger_AfterQuery_ErrNoRows_LogsDebug(t *testing.T) {
	buf := &bytes.Buffer{}
	ctx := newTestCtx(t, buf)
	logger := &databasebun.BunLogger{}

	logger.AfterQuery(ctx, queryEvent(sql.ErrNoRows))

	require.NotEmpty(t, buf.Bytes(), "expected a log entry")
	require.Equal(t, "DEBUG", logLevel(t, buf))
}

func TestBunLogger_AfterQuery_Success_LogsDebug(t *testing.T) {
	buf := &bytes.Buffer{}
	ctx := newTestCtx(t, buf)
	logger := &databasebun.BunLogger{}

	logger.AfterQuery(ctx, queryEvent(nil))

	require.NotEmpty(t, buf.Bytes(), "expected a log entry")
	require.Equal(t, "DEBUG", logLevel(t, buf))
}

func TestBunLogger_AfterQuery_ContextCanceled_WhenDebugDisabled_NoOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	conf := config.RootConfig{Log: config.LogConfig{Level: config.LogLevelWarn, Structured: true}}
	lf := log.NewLoggerFactoryWithWriter(conf, buf)
	ctx := lf.AddToCtx(context.Background())
	logger := &databasebun.BunLogger{}

	logger.AfterQuery(ctx, queryEvent(context.Canceled))

	require.Empty(t, buf.Bytes(), "expected no log output for canceled with debug disabled")
}
