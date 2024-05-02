package test

import (
	"fmt"
	"time"

	lib "github.com/southernlabs-io/go-fw/core"
)

func NewConfig(testName string) lib.Config {
	coreConfig := lib.NewCoreConfig()
	if coreConfig.Name == "" {
		coreConfig.Name = fmt.Sprintf("no_service_name_%d", time.Now().UnixMicro())
	}
	coreConfig.Name += "-" + testName

	coreConfig.Env = lib.EnvConfig{
		Name: "test",
		Type: lib.EnvTypeTest,
	}

	config := lib.Config{CoreConfig: coreConfig}
	lib.LoadConfig(coreConfig, &config)
	return config
}

func ProvideCoreConfig(config lib.Config) lib.CoreConfig {
	return config.CoreConfig
}
