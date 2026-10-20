package config_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/config"
)

func TestLoadConfig(t *testing.T) {
	type Config struct {
		config.Config

		MapConfig map[string]bool
	}

	var conf Config
	config.LoadConfig(config.GetRootConfig(), &conf, nil)
	require.Equal(t, 0, conf.HttpServer.Port)
	require.NotEmpty(t, conf.Env.Host)
	require.True(t, conf.MapConfig["key1"])
	val, present := conf.MapConfig["key2"]
	require.True(t, present)
	require.False(t, val)

	// Test overriding values
	t.Setenv("HTTPSERVER_PORT", "9090")
	t.Setenv("MAPCONFIG_KEY2", "true")

	conf = Config{}
	config.LoadConfig(config.GetRootConfig(), &conf, nil)
	require.Equal(t, 9090, conf.HttpServer.Port)
	require.True(t, conf.MapConfig["key1"])
	require.True(t, conf.MapConfig["key2"])
}

func TestLoadConfig_OverrideSliceByIndex(t *testing.T) {
	type Config struct {
		config.Config

		Tags []string
	}

	t.Setenv("TAGS_0", "alpha")
	t.Setenv("TAGS_2", "gamma")

	var conf Config
	config.LoadConfig(config.GetRootConfig(), &conf, nil)
	require.Len(t, conf.Tags, 3)
	require.Equal(t, "alpha", conf.Tags[0])
	require.Equal(t, "", conf.Tags[1])
	require.Equal(t, "gamma", conf.Tags[2])
}

func TestLoadConfig_OverrideSliceStructFieldByIndex(t *testing.T) {
	type Integration struct {
		Name    string
		Enabled bool
	}
	type Config struct {
		config.Config

		Integrations []Integration
	}

	t.Setenv("INTEGRATIONS_0_NAME", "slack")
	t.Setenv("INTEGRATIONS_0_ENABLED", "true")
	t.Setenv("INTEGRATIONS_1_NAME", "pagerduty")

	var conf Config
	config.LoadConfig(config.GetRootConfig(), &conf, nil)
	require.Len(t, conf.Integrations, 2)
	require.Equal(t, "slack", conf.Integrations[0].Name)
	require.True(t, conf.Integrations[0].Enabled)
	require.Equal(t, "pagerduty", conf.Integrations[1].Name)
	require.False(t, conf.Integrations[1].Enabled)
}

func TestLoadConfig_OverrideWholeSliceStillWorks(t *testing.T) {
	type Config struct {
		config.Config

		Tags []string
	}

	t.Setenv("TAGS", "one,two,three")

	var conf Config
	config.LoadConfig(config.GetRootConfig(), &conf, nil)
	require.Equal(t, []string{"one", "two", "three"}, conf.Tags)
}

func TestRequestLoggerConfig_DefaultsAreSecure(t *testing.T) {
	conf := config.HttpServerConfig{}

	require.Empty(t, conf.RequestLogger.QueryParameterAllowlist)
	require.Equal(t, config.RequestLoggerReferrerOmit, conf.RequestLogger.ReferrerMode)
}

func TestRequestLoggerReferrerMode_UnmarshalText(t *testing.T) {
	var mode config.RequestLoggerReferrerMode

	require.NoError(t, mode.UnmarshalText([]byte("origin_path")))
	require.Equal(t, config.RequestLoggerReferrerOriginPath, mode)
	require.NoError(t, mode.UnmarshalText(nil))
	require.Equal(t, config.RequestLoggerReferrerOmit, mode)
	require.Error(t, mode.UnmarshalText([]byte("full")))
}

func TestLoadConfig_RequestLoggerValues(t *testing.T) {
	t.Setenv("HTTPSERVER_REQUESTLOGGER_QUERYPARAMETERALLOWLIST_0", "view")
	t.Setenv("HTTPSERVER_REQUESTLOGGER_REFERRERMODE", "origin_path")

	conf := config.Config{}
	config.LoadConfig(config.GetRootConfig(), &conf, nil)

	require.Equal(t, []string{"view"}, conf.HttpServer.RequestLogger.QueryParameterAllowlist)
	require.Equal(t, config.RequestLoggerReferrerOriginPath, conf.HttpServer.RequestLogger.ReferrerMode)
}

func TestLoadConfig_RequestLoggerYAMLValues(t *testing.T) {
	var conf config.Config
	config.LoadConfig(config.GetRootConfig(), &conf, nil)

	require.Equal(t, []string{"fixture-safe"}, conf.HttpServer.RequestLogger.QueryParameterAllowlist)
	require.Equal(t, config.RequestLoggerReferrerOriginPath, conf.HttpServer.RequestLogger.ReferrerMode)
}

func TestLoadConfig_RequestLoggerRejectsInvalidReferrerMode(t *testing.T) {
	t.Setenv("HTTPSERVER_REQUESTLOGGER_REFERRERMODE", "full")

	require.Panics(t, func() {
		var conf config.Config
		config.LoadConfig(config.GetRootConfig(), &conf, nil)
	})
}

// recordingSecretsManager serves canned secrets and records which ids were requested, in order.
type recordingSecretsManager struct {
	values map[string]string
	keys   []string
}

func (m *recordingSecretsManager) GetSecret(_ context.Context, key string) (string, error) {
	m.keys = append(m.keys, "key:"+key)
	return m.values["key:"+key], nil
}

func (m *recordingSecretsManager) GetSecretVerbatim(_ context.Context, id string) (string, error) {
	m.keys = append(m.keys, "id:"+id)
	return m.values["id:"+id], nil
}

func TestLoadConfig_JSONGroupedSecrets(t *testing.T) {
	type Llm struct {
		ApiKey string `json:"apiKey"`
	}
	type Config struct {
		config.Config

		Llm Llm
	}

	// Placeholders arrive through env overrides, exactly like a deployment would set them, and the
	// implicit field name is the key the loader sees: the Go field name, or the json tag when present.
	t.Setenv("DATABASE_USER", "<secret#>")
	t.Setenv("DATABASE_PASS", "<secret#>")
	t.Setenv("DATABASE_PORT", "<secret:DB_EXTRA#/port>")
	t.Setenv("LLM_APIKEY", "<secret:llm-keys#/openai/apiKey>")
	t.Setenv("HTTPSERVER_PORT", "<secret:HTTP#/Port>")

	sm := &recordingSecretsManager{values: map[string]string{
		"key:Database": `{"User":"app","Pass":"s3cr3t"}`,
		"id:DB_EXTRA":  `{"port":6543}`,
		"id:llm-keys":  `{"openai":{"apiKey":"sk-openai"}}`,
		"id:HTTP":      `{"Port":9090}`,
	}}

	var conf Config
	config.LoadConfig(config.GetRootConfig(), &conf, sm)

	require.Equal(t, "app", conf.Database.User)
	require.Equal(t, "s3cr3t", conf.Database.Pass)
	require.Equal(t, 6543, conf.Database.Port, "JSON number lands in an int field")
	require.Equal(t, "sk-openai", conf.Llm.ApiKey)
	require.Equal(t, 9090, conf.HttpServer.Port)

	require.ElementsMatch(t, []string{"key:Database", "id:DB_EXTRA", "id:llm-keys", "id:HTTP"}, sm.keys,
		"each secret fetched exactly once; the two Database fields share one fetch")
}

func TestLoadConfig_JSONGroupedSecretsStillWorkWithPlainForms(t *testing.T) {
	t.Setenv("DATABASE_PASS", "<secret>")
	t.Setenv("DATABASE_USER", "<secret:DB_USER>")

	sm := &recordingSecretsManager{values: map[string]string{
		"key:Database.Pass": "plain",
		"id:DB_USER":        "verbatim",
	}}

	var conf config.Config
	config.LoadConfig(config.GetRootConfig(), &conf, sm)

	require.Equal(t, "plain", conf.Database.Pass)
	require.Equal(t, "verbatim", conf.Database.User)
	require.ElementsMatch(t, []string{"key:Database.Pass", "id:DB_USER"}, sm.keys)
}
