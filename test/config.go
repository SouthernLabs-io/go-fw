package test

import (
	"testing"

	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
)

func NewTestConfig(rootConf config.RootConfig) config.Config {
	conf := config.Config{RootConfig: rootConf}
	config.LoadConfig(rootConf, &conf, nil)
	return conf
}

func NewTestRootConfig(tb testing.TB) config.RootConfig {
	rootConf := config.GetRootConfig()

	// Set a name if not set
	if rootConf.Name == "" {
		rootConf.Name = "test-service"
	}
	rootConf.Name += "-" + tb.Name()

	// Force test environment
	rootConf.Env = config.EnvConfig{
		Name: "test",
		Type: config.EnvTypeTest,
	}

	// Force default log level to debug
	rootConf.Log.Level = config.LogLevelDebug

	if rootConf.Log.Levels == nil {
		rootConf.Log.Levels = make(map[string]config.LogLevel)
	}
	// Set fx logger to info if not set
	if _, ok := rootConf.Log.Levels["go.uber.org/fx"]; !ok {
		rootConf.Log.Levels["go.uber.org/fx"] = config.LogLevelInfo
	}

	return rootConf
}

var ModuleTestConfig = fx.Options(
	fx.Provide(NewTestConfig),
	fx.Provide(NewTestRootConfig),
)
