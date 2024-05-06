package core_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/test"
)

func TestGinLoggerSkipCallers(t *testing.T) {
	config := test.NewConfig(t.Name())
	config.Env.Type = core.EnvTypeSandbox
	buffer := new(bytes.Buffer)
	logger := core.NewLoggerWithWriter(config.RootConfig, "gin_logger", buffer)
	logger.SetLevel(core.LogLevelDebug)

	gin.DefaultWriter = core.NewDefaultGinWriter(logger)
	gin.DefaultErrorWriter = core.NewDefaultErrorGinWriter(logger)
	gin.SetMode(gin.DebugMode)

	ginEngine := gin.New()
	ginEngine.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	ginEngine.GET("/error", func(c *gin.Context) {
		_ = c.Error(errors.NewUnknownf("test format"))
	})

	// Force an error
	err := ginEngine.RunFd(123456789)
	require.Error(t, err)

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
		require.True(t, strings.HasPrefix(loggerName, "github.com/gin-gonic/gin"), logMsgStr)
	}
}
