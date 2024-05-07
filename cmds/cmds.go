package cmds

import (
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/slack"
	"github.com/southernlabs-io/go-fw/version"
)

type CommandRunner any

// Command interface is used to implement sub-commands in the system.
type Command interface {
	// Cmd is the actual command name to execute.
	// For example: "app:serve"
	//
	Cmd() string

	// Short returns string about short description of the command
	// the string is shown in help screen of cobra command
	Short() string

	// Setup is used to setup flags or pre-run steps for the command.
	//
	// For example,
	//  cmd.Flags().IntVarP(&r.num, "num", "n", 5, "description")
	//
	Setup(cmd *cobra.Command)

	// Run runs the command runner
	// run returns command runner which is a function with dependency
	// injected arguments.
	//
	// For example,
	//  Command{
	//   Run: func(l core.Logger) {
	// 	   l.Info("i am working")
	// 	 },
	//  }
	//
	Run() CommandRunner

	// GetFXOpts is used to return the cmd specific FX configuration
	GetFXOpts() fx.Option
}

// WrapSubCommands gives a list of sub commands
func WrapSubCommands(commands []Command) []*cobra.Command {
	var subCommands []*cobra.Command
	for _, cmd := range commands {
		subCommands = append(subCommands, WrapSubCommand(cmd))
	}
	return subCommands
}

func WrapSubCommand(cmd Command) *cobra.Command {
	wrappedCmd := &cobra.Command{
		Use:   cmd.Cmd(),
		Short: cmd.Short(),
		Run: func(c *cobra.Command, args []string) {
			logger := core.GetLoggerForType(new(Command))
			logger.Infof("Running %s", cmd.Cmd())
			opts := fx.Options(
				core.Module,
				fx.WithLogger(func(slackLoggerInterceptor *slack.FxLifecycleLoggerInterceptor) fxevent.Logger {
					return slackLoggerInterceptor
				}),
				cmd.GetFXOpts(),
				fx.Invoke(cmd.Run()),
			)
			fx.New(opts).Run()
		},
	}
	cmd.Setup(wrappedCmd)
	return wrappedCmd
}

func startTracer(conf core.Config, logger core.Logger) {
	logger.Info("Starting tracer")
	tracer.Start(
		tracer.WithDogstatsdAddress(conf.Datadog.Agent),
		tracer.WithService(conf.Name),
		tracer.WithServiceVersion(version.SemVer),
		tracer.WithRuntimeMetrics(),
	)
}

func startProfiler(conf core.Config, logger core.Logger) {
	logger.Warn("Starting profiler")
	err := profiler.Start(
		profiler.WithVersion(version.SemVer),
		profiler.WithService(conf.Name),

		profiler.WithProfileTypes(
			profiler.CPUProfile,
			profiler.HeapProfile,
			// The profiles below are disabled by default to keep overhead
			// low, but can be enabled as needed.

			// profiler.BlockProfile,
			// profiler.MutexProfile,
			// profiler.GoroutineProfile
		))
	if err != nil {
		panic(errors.NewUnknownf("could not start DataDog Profiler, error: %w", err))
	}
}
