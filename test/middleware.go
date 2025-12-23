package test

import (
	"go.uber.org/fx"

	middleware_gin "github.com/southernlabs-io/go-fw/rest_gin/middleware"
)

var FxExportMiddlewaresGin = fx.Options(
	fx.Invoke(middleware_gin.NewMiddlewares),
	middleware_gin.RequestLoggerModule,

	//Default providers
	middleware_gin.ProvideAsMiddleware(middleware_gin.NewPanicRecovery),
	middleware_gin.ProvideAsMiddleware(middleware_gin.NewErrorHandlerFx),
)
