package cmds

import (
	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/rest"
	"github.com/southernlabs-io/go-fw/worker"
)

type ServeWorkCommand struct {
	fxCommonOpts fx.Option
	fxServerOpts fx.Option
	fxWorkerOpts fx.Option
}

func NewServeWorkCommand(fxCommonOpts, fxServerOpts, fxWorkerOpts fx.Option) *ServeWorkCommand {
	return &ServeWorkCommand{
		fxCommonOpts: fxCommonOpts,
		fxServerOpts: NewServeCommand(fxServerOpts).GetFXOpts(),
		fxWorkerOpts: NewWorkCommand(fxWorkerOpts).GetFXOpts(),
	}
}

func (w *ServeWorkCommand) Cmd() string {
	return "app:serve-work"
}

func (w *ServeWorkCommand) Short() string {
	return "server and worker application"
}

func (w *ServeWorkCommand) Setup(_ *cobra.Command) {
}

func (w *ServeWorkCommand) GetFXOpts() fx.Option {
	return fx.Options(w.fxCommonOpts, w.fxServerOpts, w.fxWorkerOpts)
}

func (w *ServeWorkCommand) Run() CommandRunner {
	return func(dep struct {
		fx.In

		Conf          core.Config
		HTTPHandler   rest.HTTPHandler       //It is here for the container to initialize it
		WorkerHandler []worker.WorkerHandler `group:"worker_handlers"` //It is here for the container to initialize it
	}) {
		logger := core.GetLoggerForType(w)
		if dep.Conf.Datadog.Tracing {
			startTracer(dep.Conf, logger)
		}
		if dep.Conf.Datadog.Profiling {
			startProfiler(dep.Conf, logger)
		}
	}
}
