package config_test

import (
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
