package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/rest"
	"github.com/southernlabs-io/go-fw/rest/middleware"
	"github.com/southernlabs-io/go-fw/rest/providers"
)

type ServeCommand struct {
	fxOpts fx.Option
}

func NewServeCommand(fxOpts fx.Option) *ServeCommand {
	return &ServeCommand{fxOpts: fx.Options(fxOpts, providers.Module, middleware.Module, rest.Module)}
}

func (s *ServeCommand) Cmd() string {
	return "app:serve"
}

func (s *ServeCommand) Short() string {
	return "serve application"
}

func (s *ServeCommand) Setup(_ *cobra.Command) {
}

func (s *ServeCommand) GetFXOpts() fx.Option {
	return s.fxOpts
}

func (s *ServeCommand) Run() CommandRunner {
	return func(dep struct {
		fx.In

		Conf        config.Config
		HTTPHandler rest.HTTPHandler //It is here for the container to initialize it
	}) {
		logger := log.GetLoggerForType(s)
		if dep.Conf.Datadog.Tracing {
			startTracer(dep.Conf, logger)
		}
		if dep.Conf.Datadog.Profiling {
			startProfiler(dep.Conf, logger)
		}
	}
}
