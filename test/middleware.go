package test

import (
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/rest_gin/middleware"
)

var FxExportMiddlewares = fx.Options(
	fx.Invoke(middleware.NewMiddlewares),
	middleware.RequestLoggerModule,

	//Default providers
	middleware.ProvideAsMiddleware(middleware.NewPanicRecovery),
	middleware.ProvideAsMiddleware(middleware.NewErrorHandlerFx),
)
