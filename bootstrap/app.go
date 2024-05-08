package bootstrap

import (
	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/cmd"
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

func NewApp(commands ...cmd.Command) *cobra.Command {
	rootCmd.AddCommand(cmd.WrapSubCommands(commands)...)
	return rootCmd
}

func NewAppWithServe(fxOpts fx.Option) *cobra.Command {
	return NewApp(cmd.NewServeCommand(fxOpts))
}

func NewAppWithWork(fxOpts fx.Option) *cobra.Command {
	return NewApp(cmd.NewWorkCommand(fxOpts))
}

func NewAppWithServeAndWork(fxCommonOpts, fxServerOpts, fxWorkerOpts fx.Option) *cobra.Command {
	return NewApp(
		cmd.NewServeCommand(fx.Options(fxCommonOpts, fxServerOpts)),
		cmd.NewWorkCommand(fx.Options(fxCommonOpts, fxWorkerOpts)),
		cmd.NewServeWorkCommand(fxCommonOpts, fxServerOpts, fxWorkerOpts),
	)
}
