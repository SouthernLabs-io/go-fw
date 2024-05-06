package providers

import (
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/middlewares"
)

func ProvideAsHealthCheck(provider any, anns ...fx.Annotation) fx.Option {
	return core.FxProvideAs[middlewares.HealthCheckProvider](
		provider,
		anns,
		[]fx.Annotation{fx.ResultTags(`group:"health_checks"`)},
	)
}

func ProvideAsAuthN(provider any, anns ...fx.Annotation) fx.Option {
	return core.FxProvideAs[middlewares.AuthNProvider](
		provider,
		anns,
		nil,
	)
}

func ProvideAsAuthZ(provider any, anns ...fx.Annotation) fx.Option {
	return core.FxProvideAs[middlewares.AuthZProvider](
		provider,
		anns,
		nil,
	)
}

var Module = fx.Options(
	ProvideAsHealthCheck(
		NewDatabaseHealthCheckProvider,
		fx.ParamTags(`optional:"true"`),
	),
	ProvideAsHealthCheck(
		NewRedisHealthCheckProvider,
		fx.ParamTags(`optional:"true"`),
	),
)
