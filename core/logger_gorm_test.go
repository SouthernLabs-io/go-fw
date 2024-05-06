package core_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/test"
)

func TestGormLoggerSkipCallers(t *testing.T) {
	test.IntegrationTest(t)

	config := test.NewConfig(t.Name())
	config.Env.Type = core.EnvTypeSandbox
	buffer := new(bytes.Buffer)
	logger := core.NewLoggerWithWriter(config.RootConfig, "gorm_test", buffer)
	logger.SetLevel(core.LogLevelTrace)

	_, err := gorm.Open(postgres.Open(""), &gorm.Config{
		Logger: core.NewGormLogger(logger),
	})
	require.Error(t, err)

	gormDB, err := gorm.Open(
		postgres.Open("user=postgres password=postgres dbname=postgres host=localhost"),
		&gorm.Config{
			Logger: core.NewGormLogger(logger),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, gormDB)
	require.NoError(t, gormDB.Exec("DROP TABLE IF EXISTS post;").Error)
	require.NoError(t, gormDB.Exec("CREATE TABLE post (id int NOT NULL, title text, body text, PRIMARY KEY(id));").Error)
	require.NoError(t, gormDB.Exec("DROP TABLE IF EXISTS post;").Error)
	sqlDB, err := gormDB.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

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
		require.True(t,
			strings.HasPrefix(loggerName, "gorm.io/gorm") ||
				strings.HasPrefix(loggerName, "github.com/southernlabs-io/go-fw/core_test"),
			logMsgStr,
		)
	}
}

func TestGormLoggerLogMode(t *testing.T) {
	test.IntegrationTest(t)

	config := test.NewConfig(t.Name())
	config.Env.Type = core.EnvTypeSandbox
	buffer := new(bytes.Buffer)
	logger := core.NewLoggerWithWriter(config.RootConfig, "gorm_test", buffer)
	logger.SetLevel(core.LogLevelWarn)

	gormDB, err := gorm.Open(
		postgres.Open("user=postgres password=postgres dbname=postgres host=localhost sslmode=disable"),
		&gorm.Config{
			Logger: core.NewGormLogger(logger),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, gormDB)
	require.Empty(t, buffer)

	// Issue a query, it should not produce any log messages
	require.NoError(t, gormDB.Exec("SELECT 1;").Error)
	require.Empty(t, buffer)

	// Enable gorm Debug mode
	gormDBDebug := gormDB.Debug()
	require.NoError(t, gormDBDebug.Exec("SELECT 1;").Error)
	require.Greater(t, buffer.Len(), 0, "Expected log messages")
}
