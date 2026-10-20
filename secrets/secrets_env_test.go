package secrets_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/secrets"
)

func TestEnvSecretsManager_JSONGroupedSecrets(t *testing.T) {
	rootConf := config.GetRootConfig()
	rootConf.Env.Name = "local"
	// Keep the transformed id a valid, collision-free env var name: "GROUPED_local_Database".
	rootConf.Secrets.PrefixFmt = "GROUPED_{Env.Name}_"

	// One env var holds the whole group; the manager falls back to the upper-cased name.
	t.Setenv("GROUPED_LOCAL_DATABASE", `{"User":"app","Pass":"s3cr3t","Port":6543}`)
	t.Setenv("LLM_KEYS", `{"openai":{"apiKey":"sk-openai"}}`)

	// Placeholders arrive as config overrides, as a .env file would provide them.
	t.Setenv("DATABASE_USER", "<secret#>")
	t.Setenv("DATABASE_PASS", "<secret#>")
	t.Setenv("DATABASE_PORT", "<secret#>")
	t.Setenv("APIKEY", "<secret:LLM_KEYS#/openai/apiKey>")
	t.Setenv("HTTPSERVER_PORT", "8080")

	sm := secrets.NewEnvSecretsManager(rootConf, nil)

	type Config struct {
		config.Config

		ApiKey string
	}

	var conf Config
	config.LoadConfig(rootConf, &conf, sm)

	require.Equal(t, "app", conf.Database.User)
	require.Equal(t, "s3cr3t", conf.Database.Pass)
	require.Equal(t, 6543, conf.Database.Port)
	require.Equal(t, "sk-openai", conf.ApiKey)
	require.Equal(t, 8080, conf.HttpServer.Port)
}

func TestEnvSecretsManager_PlainFormsUnchanged(t *testing.T) {
	rootConf := config.GetRootConfig()
	rootConf.Env.Name = "local"
	rootConf.Secrets.PrefixFmt = "PLAIN_{Env.Name}_"

	t.Setenv("PLAIN_LOCAL_DATABASE.PASS", "by-path")
	t.Setenv("DB_USER", "verbatim")

	sm := secrets.NewEnvSecretsManager(rootConf, nil)

	got, err := sm.GetSecret(context.Background(), "Database.Pass")
	require.NoError(t, err)
	require.Equal(t, "by-path", got)

	got, err = sm.GetSecretVerbatim(context.Background(), "DB_USER")
	require.NoError(t, err)
	require.Equal(t, "verbatim", got)

	_, err = sm.GetSecretVerbatim(context.Background(), "MISSING_SECRET")
	require.ErrorContains(t, err, "environment variable MISSING_SECRET not set")
}
