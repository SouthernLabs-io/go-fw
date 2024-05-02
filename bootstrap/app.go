package bootstrap

import (
	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/cmds"
)

var rootCmd = &cobra.Command{
	Use:   "",
	Short: "Go service",
	Long: `
	This is a command runner or cli for service architecture in golang.
	It is build with uber-go/fx, gin-gonic/gin and based on dipeshdulal/clean-gin. 
	`,
	TraverseChildren: true,
}

func NewApp(commands ...cmds.Command) *cobra.Command {
	rootCmd.AddCommand(cmds.WrapSubCommands(commands)...)
	return rootCmd
}

func NewAppWithServe(fxOpts fx.Option) *cobra.Command {
	return NewApp(cmds.NewServeCommand(fxOpts))
}

func NewAppWithWork(fxOpts fx.Option) *cobra.Command {
	return NewApp(cmds.NewWorkCommand(fxOpts))
}

func NewAppWithServeAndWork(fxCommonOpts, fxServerOpts, fxWorkerOpts fx.Option) *cobra.Command {
	return NewApp(
		cmds.NewServeCommand(fx.Options(fxCommonOpts, fxServerOpts)),
		cmds.NewWorkCommand(fx.Options(fxCommonOpts, fxWorkerOpts)),
		cmds.NewServeWorkCommand(fxCommonOpts, fxServerOpts, fxWorkerOpts),
	)
}
