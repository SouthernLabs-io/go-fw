package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/worker"
)

type WorkCommand struct {
	fxOpts fx.Option
}

func NewWorkCommand(fxOpts fx.Option) *WorkCommand {
	return &WorkCommand{fxOpts: fx.Options(fxOpts, worker.ModuleWorkerHandler)}
}

func (w *WorkCommand) Cmd() string {
	return "app:work"
}

func (w *WorkCommand) Short() string {
	return "worker application"
}

func (w *WorkCommand) Setup(_ *cobra.Command) {
}

func (w *WorkCommand) GetFXOpts() fx.Option {
	return w.fxOpts
}

func (w *WorkCommand) Run() CommandRunner {
	return func(dep struct {
		fx.In

		Conf          config.Config
		WorkerHandler []worker.Handler `group:"worker_handlers"` //It is here for the container to initialize it
	}) {
		logger := log.GetLoggerForType(w)
		if dep.Conf.Datadog.Tracing {
			startTracer(dep.Conf, logger)
		}
		if dep.Conf.Datadog.Profiling {
			startProfiler(dep.Conf, logger)
		}
	}
}
