package main

import (
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/bootstrap"
	lib "github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/middlewares"
	"github.com/southernlabs-io/go-fw/rest"
)

func main() {
	defer lib.DeferredPanicToLogAndExit()
	var deps = fx.Options(
		// middlewares
		middlewares.RequestLoggerModule,

		rest.ProvideAsResource(NewMyResource),
	)
	err := bootstrap.NewAppWithServe(deps).Execute()
	if err != nil {
		panic(errors.NewUnknownf("failed to execute with error: %w", err))
	}
}
