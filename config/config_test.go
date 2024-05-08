package config_test

import (
	"os"
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
	config.LoadConfig(config.GetCoreConfig(), &conf, nil)
	require.Equal(t, 8080, conf.HttpServer.Port)
	require.NotEmpty(t, conf.Env.Host)
	require.True(t, conf.MapConfig["key1"])
	val, present := conf.MapConfig["key2"]
	require.True(t, present)
	require.False(t, val)

	// Test overriding values
	err := os.Setenv("MAPCONFIG_KEY2", "true")
	require.NoError(t, err)
	err = os.Setenv("HTTPSERVER_PORT", "9090")
	require.NoError(t, err)

	conf = Config{}
	config.LoadConfig(config.GetCoreConfig(), &conf, nil)
	require.Equal(t, 9090, conf.HttpServer.Port)
	require.True(t, conf.MapConfig["key1"])
	require.True(t, conf.MapConfig["key2"])
}
