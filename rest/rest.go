package rest

import (
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/rest/middleware"
)

var Module = fx.Options(
	middleware.Module,
	fx.Invoke(NewResources),
	fx.Provide(NewStdServer),
)
