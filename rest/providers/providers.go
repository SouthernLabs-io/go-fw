package providers

import (
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/di"
	"github.com/southernlabs-io/go-fw/rest/middleware"
)

func ProvideAsHealthCheck(provider any, anns ...fx.Annotation) fx.Option {
	return di.FxProvideAs[middleware.HealthCheckProvider](
		provider,
		anns,
		[]fx.Annotation{fx.ResultTags(`group:"health_checks"`)},
	)
}

func ProvideAsAuthN(provider any, anns ...fx.Annotation) fx.Option {
	return di.FxProvideAs[middleware.AuthNProvider](
		provider,
		anns,
		nil,
	)
}

func ProvideAsAuthZ(provider any, anns ...fx.Annotation) fx.Option {
	return di.FxProvideAs[middleware.AuthZProvider](
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
