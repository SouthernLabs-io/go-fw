package main

import (
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/bootstrap"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/panics"
	"github.com/southernlabs-io/go-fw/rest"
	"github.com/southernlabs-io/go-fw/rest/middleware"
)

func main() {
	defer panics.DeferredPanicToLogAndExit()
	var deps = fx.Options(

		// Service configuration
		fx.Provide(NewConfig),

		// Middlewares
		middleware.RequestLoggerModule,

		// Register Resources
		rest.ProvideAsResource(NewMyResource),
	)
	err := bootstrap.NewAppWithServe(deps).Execute()
	if err != nil {
		panic(errors.NewUnknownf("failed to execute with error: %w", err))
	}
}
