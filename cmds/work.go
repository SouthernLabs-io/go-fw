package cmds

import (
	"github.com/spf13/cobra"
	"go.uber.org/fx"

	lib "github.com/southernlabs-io/go-fw/core"
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

		Conf          lib.Config
		WorkerHandler []worker.WorkerHandler `group:"worker_handlers"` //It is here for the container to initialize it
	}) {
		logger := lib.GetLoggerForType(w)
		if dep.Conf.Datadog.Tracing {
			startTracer(dep.Conf, logger)
		}
		if dep.Conf.Datadog.Profiling {
			startProfiler(dep.Conf, logger)
		}
	}
}
