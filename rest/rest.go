package rest

import (
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/rest/middleware"
)

var FxExport = fx.Options(
	middleware.FxExport,
	fx.Provide(NewResources),
	fx.Provide(NewStdServer),
)
