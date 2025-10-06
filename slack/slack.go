package slack

import "go.uber.org/fx"

var FxExport = fx.Options(
	fx.Provide(NewSlackClient),
	fx.Decorate(NewFxLoggerFactory),
)
