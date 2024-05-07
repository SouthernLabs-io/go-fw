package slack

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(NewSlackClient),
	fx.Provide(NewSlackFxLifecycleLoggerInterceptor),
)
