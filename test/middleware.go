package test

import (
	"go.uber.org/fx"

	middlewares2 "github.com/southernlabs-io/go-fw/rest/middlewares"
)

var TestModuleMiddlewares = fx.Options(
	fx.Invoke(middlewares2.NewMiddlewares),
	middlewares2.RequestLoggerModule,

	//Default providers
	middlewares2.ProvideAsMiddleware(middlewares2.NewPanicRecovery),
	middlewares2.ProvideAsMiddleware(middlewares2.NewErrorHandlerFx),
)
