package database_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"

	_ "github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/database"
	"github.com/southernlabs-io/go-fw/test"
)

func TestGooseLoggerSkipCallers(t *testing.T) {
	test.IntegrationTest(t)

	config := test.NewConfig(t.Name())
	config.Env.Type = core.EnvTypeSandbox
	buffer := new(bytes.Buffer)
	logger := core.NewLoggerWithWriter(config.RootConfig, "goose_logger", buffer)
	logger.SetLevel(core.LogLevelDebug)

	gooseLogger := database.NewGooseLogger(logger)
	goose.SetLogger(gooseLogger)
	sqlDB, err := sql.Open(
		"pgx",
		"user=postgres password=postgres dbname=postgres host=localhost",
	)
	require.NoError(t, err)

	fs := fstest.MapFS{
		"1_test_migration.sql": {Data: []byte(`
-- +goose Up
CREATE TABLE post (
	id int NOT NULL,
	title text,
	body text,
	PRIMARY KEY(id)
);

-- +goose Down
DROP TABLE IF EXISTS post;`,
		)},
	}
	goose.SetBaseFS(fs)
	_, err = goose.EnsureDBVersion(sqlDB)
	require.NoError(t, err)
	err = goose.Reset(sqlDB, ".")
	require.NoError(t, err)
	err = goose.Up(sqlDB, ".")
	require.NoError(t, err)

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
		require.True(t, strings.HasPrefix(loggerName, "github.com/pressly/goose/v3"), logMsgStr)
	}
}
