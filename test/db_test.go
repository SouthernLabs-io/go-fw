package test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/config"
)

func TestCreateTestDBName_LongNameStaysWithinPostgresLimit(t *testing.T) {
	conf := config.Config{
		RootConfig: config.RootConfig{
			Name: strings.Repeat("very-long-integration-test-name-", 6),
			Env:  config.EnvConfig{Name: "test"},
		},
	}

	dbName := CreateTestDBName(conf)

	require.LessOrEqual(t, len(dbName), 63)
	require.NotContains(t, dbName, "%!(BADPREC)")
	require.NotContains(t, dbName, "%!")
	require.True(t, strings.HasSuffix(dbName, "_test"))
}

func TestCreateTestDBName_UsesHashWhenPrefixIsFullyTrimmed(t *testing.T) {
	conf := config.Config{
		RootConfig: config.RootConfig{
			Name: "test",
			Env:  config.EnvConfig{Name: "test"},
		},
	}

	dbName := CreateTestDBName(conf)

	require.NotEmpty(t, dbName)
	require.LessOrEqual(t, len(dbName), 63)
	require.True(t, strings.HasSuffix(dbName, "_test"))
	require.Len(t, strings.TrimSuffix(dbName, "_test"), 16)
}
