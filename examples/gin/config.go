package main

import "github.com/southernlabs-io/go-fw/config"

type Config struct {
	config.Config

	Mapping map[string]any
}

func NewConfig(rootConf config.RootConfig) Config {
	var conf Config
	config.LoadConfig(rootConf, &conf, nil)
	return conf
}
