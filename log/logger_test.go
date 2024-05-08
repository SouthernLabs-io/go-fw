package log_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/database"
	"github.com/southernlabs-io/go-fw/log"
)

type TheStruct struct {
}

func (t TheStruct) Method(lf *log.LoggerFactory) log.Logger {
	return lf.GetLogger()
}

type WrapperStruct struct {
	TheStruct
}

func (t *WrapperStruct) CallWrappedMethod(lf *log.LoggerFactory) log.Logger {
	return t.TheStruct.Method(lf)
}

func (t *WrapperStruct) MethodInPtr(lf *log.LoggerFactory) log.Logger {
	return lf.GetLogger()
}

func TestLevels(t *testing.T) {
	logConf := config.LogConfig{
		Level: config.LogLevelError,
		Levels: map[string]config.LogLevel{
			"github.com/southernlabs-io":                                             config.LogLevelWarn,
			"github.com/southernlabs-io/go-fw":                                       config.LogLevelInfo,
			"github.com/southernlabs-io/go-fw/database":                              config.LogLevelDebug,
			"github.com/southernlabs-io/go-fw/log":                                   config.LogLevelDebug,
			"github.com/southernlabs-io/go-fw/secret.AWSSecretsManager":              config.LogLevelTrace,
			"github.com/southernlabs-io/go-fw/log_test":                              config.LogLevelDebug,
			"github.com/southernlabs-io/go-fw/log_test.TestLevels":                   config.LogLevelTrace,
			"github.com/southernlabs-io/go-fw/log_test.CustomSlice":                  config.LogLevelTrace,
			"github.com/southernlabs-io/go-fw/log_test.TheStruct.Method":             config.LogLevelTrace,
			"github.com/southernlabs-io/go-fw/log_test.(*WrapperStruct).MethodInPtr": config.LogLevelTrace,
			"my-local-package":      config.LogLevelInfo,
			"my-local-package/math": config.LogLevelDebug,
		},
	}
	lf := log.NewLoggerFactory(config.RootConfig{
		Log: logConf,
	})
	require.NotNil(t, lf)

	var l log.Logger

	// Test Root level
	l = lf.GetLoggerForPath("/")
	require.NotZero(t, l)
	require.Equal(t, config.LogLevelError, l.Level())

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
	require.Equal(t, config.LogLevelWarn, l.Level())

	// Test case-sensitive match
	l = lf.GetLoggerForPath("github.com/southernlabs-io/go-FW")
	require.NotZero(t, l)
	require.Equal(t, config.LogLevelWarn, l.Level())

	// Test level by segment match at package level
	l = lf.GetLoggerForPath("github.com/southernlabs-io/go-fw/another-package")
	require.NotZero(t, l)
	require.Equal(t, config.LogLevelInfo, l.Level())

	// Test level by segment match at package function level
	l = lf.GetLoggerForPath("github.com/southernlabs-io/go-fw/log.AnotherFunction")
	require.NotZero(t, l)
	require.Equal(t, config.LogLevelDebug, l.Level())

	// Test level by type exact match
	l = lf.GetLoggerForType(secrets.AWSSecretsManager{})
	require.NotZero(t, l)
	require.Equal(t, config.LogLevelTrace, l.Level())

	// Test level by type package match
	l = lf.GetLoggerForType(database.DB{})
	require.NotZero(t, l)
	require.Equal(t, config.LogLevelDebug, l.Level())

	// Test level by stack match
	l = lf.GetLogger()
	require.NotZero(t, l)
	require.Equal(t, config.LogLevelTrace, l.Level())

	// Test level by function match from inline call
	func() {
		l = lf.GetLogger()
		require.NotZero(t, l)
		require.Equal(t, config.LogLevelTrace, l.Level())
	}()

	// Test level by exact match from inline type def
	type CustomSlice []string
	l = lf.GetLoggerForType(CustomSlice{})
	require.NotZero(t, l)
	require.Equal(t, config.LogLevelTrace, l.Level())

	// Test level by package match from inline type def
	type CustomString string
	l = lf.GetLoggerForType(CustomString(""))
	require.NotZero(t, l)
	require.Equal(t, config.LogLevelDebug, l.Level())

	// Test level by exact match from struct method
	l = TheStruct{}.Method(lf)
	require.NotZero(t, l)
	require.Equal(t, config.LogLevelTrace, l.Level())

	// Test level by exact match from struct method
	l = (&TheStruct{}).Method(lf)
	require.NotZero(t, l)
	require.Equal(t, config.LogLevelTrace, l.Level())

	// Test level by exact match from struct method
	l = (&WrapperStruct{}).MethodInPtr(lf)
	require.NotZero(t, l)
	require.Equal(t, config.LogLevelTrace, l.Level())

	// Test level by exact match from struct method
	l = (&WrapperStruct{}).Method(lf)
	require.NotZero(t, l)
	require.Equal(t, config.LogLevelTrace, l.Level())

	l = (&WrapperStruct{}).CallWrappedMethod(lf)
	require.NotZero(t, l)
	require.Equal(t, config.LogLevelTrace, l.Level())
}

func TestContext(t *testing.T) {
	lf := log.NewLoggerFactory(config.RootConfig{
		Log: config.LogConfig{
			Level:  config.LogLevelDebug,
			Writer: config.LogConfigWriterBuffer,
		},
	})
	require.NotNil(t, lf)

	// Test with empty context
	ctx := context.Background()
	l := lf.GetLoggerFromCtx(ctx)
	require.NotZero(t, l)
	require.Equal(t, config.LogLevelDebug, l.Level())
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
	ctx = log.CtxWithLoggerAttrs(ctx,
		slog.String("req_id", "123"),
		slog.String("method", "GET"),
	)
	l = lf.GetLoggerFromCtx(ctx)
	require.NotZero(t, l)
	require.Equal(t, config.LogLevelDebug, l.Level())
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
	ctx = log.CtxAppendLoggerAttrs(ctx, slog.Int64("duration", time.Second.Milliseconds()))
	l = lf.GetLoggerFromCtx(ctx)
	require.NotZero(t, l)
	require.Equal(t, config.LogLevelDebug, l.Level())
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
