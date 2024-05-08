package test

import (
	"fmt"
	"time"

	"github.com/southernlabs-io/go-fw/config"
)

func NewConfig(testName string) config.Config {
	coreConfig := config.GetCoreConfig()
	if coreConfig.Name == "" {
		coreConfig.Name = fmt.Sprintf("no_service_name_%d", time.Now().UnixMicro())
	}
	coreConfig.Name += "-" + testName

	// Set default test config, it can be overridden in the test config.yaml
	coreConfig.Env = config.EnvConfig{
		Name: "test",
		Type: config.EnvTypeTest,
	}
	coreConfig.Log.Level = config.LogLevelDebug

	conf := config.Config{RootConfig: coreConfig}
	config.LoadConfig(coreConfig, &conf, nil)
	return conf
}

func ProvideCoreConfig(config config.Config) config.RootConfig {
	return config.RootConfig
}
