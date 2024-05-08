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

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/di"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/test"
)

func TestFxLoggerSkipCallers(t *testing.T) {
	conf := test.NewConfig(t.Name())
	conf.Env.Type = config.EnvTypeSandbox
	buffer := new(bytes.Buffer)
	logger := log.NewLoggerWithWriter(conf.RootConfig, "fx_logger", buffer)
	logger.SetLevel(config.LogLevelDebug)
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
		methodName := loggerMap["method_name"].(string)
		require.True(t, strings.HasPrefix(methodName, "go.uber.org/fx."), methodName)
		// The logBuffer is a private type that wraps the logger, so it should not be considered a caller
		require.False(t, strings.HasPrefix(methodName, "go.uber.org/fx.(*logBuffer)"), methodName)
	}
}
