package main

import (
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/bootstrap"
	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/rest"
	"github.com/southernlabs-io/go-fw/rest/middlewares"
)

func main() {
	defer core.DeferredPanicToLogAndExit()
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
