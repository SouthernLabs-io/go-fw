package core_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/database"
)

type TheStruct struct {
}

func (t TheStruct) Method(lf *core.LoggerFactory) core.Logger {
	return lf.GetLogger()
}

type WrapperStruct struct {
	TheStruct
}

func (t *WrapperStruct) CallWrappedMethod(lf *core.LoggerFactory) core.Logger {
	return t.TheStruct.Method(lf)
}

func (t *WrapperStruct) MethodInPtr(lf *core.LoggerFactory) core.Logger {
	return lf.GetLogger()
}

func TestLevels(t *testing.T) {
	logConf := core.LogConfig{
		Level: core.LogLevelError,
		Levels: map[string]core.LogLevel{
			"github.com/southernlabs-io":                                              core.LogLevelWarn,
			"github.com/southernlabs-io/go-fw":                                        core.LogLevelInfo,
			"github.com/southernlabs-io/go-fw/database":                               core.LogLevelDebug,
			"github.com/southernlabs-io/go-fw/core":                                   core.LogLevelDebug,
			"github.com/southernlabs-io/go-fw/core.AWSSecretsManager":                 core.LogLevelTrace,
			"github.com/southernlabs-io/go-fw/core_test":                              core.LogLevelDebug,
			"github.com/southernlabs-io/go-fw/core_test.TestLevels":                   core.LogLevelTrace,
			"github.com/southernlabs-io/go-fw/core_test.CustomSlice":                  core.LogLevelTrace,
			"github.com/southernlabs-io/go-fw/core_test.TheStruct.Method":             core.LogLevelTrace,
			"github.com/southernlabs-io/go-fw/core_test.(*WrapperStruct).MethodInPtr": core.LogLevelTrace,
			"my-local-package":      core.LogLevelInfo,
			"my-local-package/math": core.LogLevelDebug,
		},
	}
	lf := core.NewLoggerFactory(core.RootConfig{
		Log: logConf,
	})
	require.NotNil(t, lf)

	var l core.Logger

	// Test Root level
	l = lf.GetLoggerForPath("/")
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelError, l.Level())

	// Test level by exact math
	for pth, level := range logConf.Levels {
		l = lf.GetLoggerForPath(pth)
		require.NotZero(t, l)
		require.Equal(t, level, l.Level())
	}

	// Test allocations do not change after multiple calls to the same path
	allocs := testing.AllocsPerRun(100, func() {
		lf.GetLoggerForPath("github.com/southernlabs-io/go-fw")
	})
	require.EqualValues(t, 0, allocs)

	// Test no partial match
	l = lf.GetLoggerForPath("github.com/southernlabs-io/go-")
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelWarn, l.Level())

	// Test case-sensitive match
	l = lf.GetLoggerForPath("github.com/southernlabs-io/go-FW")
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelWarn, l.Level())

	// Test level by segment match at package level
	l = lf.GetLoggerForPath("github.com/southernlabs-io/go-fw/another-package")
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelInfo, l.Level())

	// Test level by segment match at package function level
	l = lf.GetLoggerForPath("github.com/southernlabs-io/go-fw/core.AnotherFunction")
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelDebug, l.Level())

	// Test level by type exact match
	l = lf.GetLoggerForType(core.AWSSecretsManager{})
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelTrace, l.Level())

	// Test level by type package match
	l = lf.GetLoggerForType(database.DB{})
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelDebug, l.Level())

	// Test level by stack match
	l = lf.GetLogger()
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelTrace, l.Level())

	// Test level by function match from inline call
	func() {
		l = lf.GetLogger()
		require.NotZero(t, l)
		require.Equal(t, core.LogLevelTrace, l.Level())
	}()

	// Test level by exact match from inline type def
	type CustomSlice []string
	l = lf.GetLoggerForType(CustomSlice{})
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelTrace, l.Level())

	// Test level by package match from inline type def
	type CustomString string
	l = lf.GetLoggerForType(CustomString(""))
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelDebug, l.Level())

	// Test level by exact match from struct method
	l = TheStruct{}.Method(lf)
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelTrace, l.Level())

	// Test level by exact match from struct method
	l = (&TheStruct{}).Method(lf)
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelTrace, l.Level())

	// Test level by exact match from struct method
	l = (&WrapperStruct{}).MethodInPtr(lf)
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelTrace, l.Level())

	// Test level by exact match from struct method
	l = (&WrapperStruct{}).Method(lf)
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelTrace, l.Level())

	l = (&WrapperStruct{}).CallWrappedMethod(lf)
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelTrace, l.Level())
}

func TestContext(t *testing.T) {
	lf := core.NewLoggerFactory(core.RootConfig{
		Log: core.LogConfig{
			Level:  core.LogLevelDebug,
			Writer: core.LogConfigWriterBuffer,
		},
	})
	require.NotNil(t, lf)

	// Test with empty context
	ctx := context.Background()
	l := lf.GetLoggerFromCtx(ctx)
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelDebug, l.Level())
	buffer, isBuffer := l.Writer().(*bytes.Buffer)
	require.True(t, isBuffer)
	l.Info("an info message")
	require.Greater(t, buffer.Len(), 0)
	var msg map[string]any
	require.NoError(t, json.Unmarshal(buffer.Bytes(), &msg))
	require.Contains(t, msg, "msg")
	require.Equal(t, msg["msg"], "an info message")
	require.NotContains(t, msg, "req_id")
	require.NotContains(t, msg, "duration")

	// Test adding context attributes
	ctx = core.CtxWithLoggerAttrs(ctx,
		slog.String("req_id", "123"),
		slog.String("method", "GET"),
	)
	l = lf.GetLoggerFromCtx(ctx)
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelDebug, l.Level())
	buffer, isBuffer = l.Writer().(*bytes.Buffer)
	require.True(t, isBuffer)
	buffer.Reset()
	l.Info("an info message")
	require.Greater(t, buffer.Len(), 0)
	require.NoError(t, json.Unmarshal(buffer.Bytes(), &msg))
	require.Contains(t, msg, "msg")
	require.Equal(t, msg["msg"], "an info message")
	require.Contains(t, msg, "req_id")
	require.Equal(t, "123", msg["req_id"])
	require.Contains(t, msg, "method")
	require.Equal(t, "GET", msg["method"])

	// Test adding context attributes with append
	ctx = core.CtxAppendLoggerAttrs(ctx, slog.Int64("duration", time.Second.Milliseconds()))
	l = lf.GetLoggerFromCtx(ctx)
	require.NotZero(t, l)
	require.Equal(t, core.LogLevelDebug, l.Level())
	buffer, isBuffer = l.Writer().(*bytes.Buffer)
	require.True(t, isBuffer)
	buffer.Reset()
	l.Info("an info message")
	require.Greater(t, buffer.Len(), 0)
	require.NoError(t, json.Unmarshal(buffer.Bytes(), &msg))
	require.Contains(t, msg, "msg")
	require.Equal(t, msg["msg"], "an info message")
	require.Contains(t, msg, "req_id")
	require.Equal(t, "123", msg["req_id"])
	require.Contains(t, msg, "method")
	require.Equal(t, "GET", msg["method"])
	require.Contains(t, msg, "duration")
	require.Equal(t, float64(time.Second.Milliseconds()), msg["duration"])
}
