package di_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/fxtest"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/di"
	"github.com/southernlabs-io/go-fw/test"
)

func TestFxLoggerSkipCallers(t *testing.T) {
	config := test.NewConfig(t.Name())
	config.Env.Type = core.EnvTypeSandbox
	buffer := new(bytes.Buffer)
	logger := core.NewLoggerWithWriter(config.RootConfig, "fx_logger", buffer)
	logger.SetLevel(core.LogLevelDebug)
	fxLogger := di.NewFxLogger(logger)

	fxApp := fxtest.New(t, fx.WithLogger(func() fxevent.Logger { return fxLogger }))
	fxApp.RequireStart().RequireStop()
	require.Greater(t, buffer.Len(), 0, "Expected log messages")
	var logMsg map[string]any
	for _, logMsgStr := range strings.Split(buffer.String(), "\n") {
		if logMsgStr == "" {
			continue
		}
		t.Log("logMsg: ", logMsgStr)
		require.NoError(t, json.Unmarshal([]byte(logMsgStr), &logMsg))
		loggerMap, isMap := logMsg["logger"].(map[string]any)
		require.True(t, isMap)
		loggerName := loggerMap["method_name"].(string)
		require.True(t, strings.HasPrefix(loggerName, "go.uber.org/fx."), loggerName)
		// The logBuffer is a private type that wraps the logger, so it should not be considered a caller
		require.False(t, strings.HasPrefix(loggerName, "go.uber.org/fx.(*logBuffer)"), loggerName)
	}
}
