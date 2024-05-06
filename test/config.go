package test

import (
	"fmt"
	"time"

	"github.com/southernlabs-io/go-fw/core"
)

func NewConfig(testName string) core.Config {
	coreConfig := core.NewCoreConfig()
	if coreConfig.Name == "" {
		coreConfig.Name = fmt.Sprintf("no_service_name_%d", time.Now().UnixMicro())
	}
	coreConfig.Name += "-" + testName

	coreConfig.Env = core.EnvConfig{
		Name: "test",
		Type: core.EnvTypeTest,
	}

	config := core.Config{RootConfig: coreConfig}
	core.LoadConfig(coreConfig, &config)
	return config
}

func ProvideCoreConfig(config core.Config) core.RootConfig {
	return config.RootConfig
}
