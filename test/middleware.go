package test

import (
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/middlewares"
)

var TestModuleMiddlewares = fx.Options(
	fx.Invoke(middlewares.NewMiddlewares),
	middlewares.RequestLoggerModule,

	//Default providers
	middlewares.ProvideAsMiddleware(middlewares.NewPanicRecovery),
	middlewares.ProvideAsMiddleware(middlewares.NewErrorHandlerFx),
)
