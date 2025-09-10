package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/rest"
)

type ServeCommand struct {
	fxOpts fx.Option
}

func NewServeCommand(fxOpts fx.Option) *ServeCommand {
	return &ServeCommand{fxOpts: fx.Options(fxOpts, rest.Module)}
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

		Conf config.Config
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
