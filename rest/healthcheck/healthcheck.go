package healthcheck

import (
	"github.com/southernlabs-io/go-fw/di"
	"go.uber.org/fx"
)

type Provider interface {
	GetName() string
	HealthCheck() error
}

func ProvideAsHealthCheckProvider(provider any, anns ...fx.Annotation) fx.Option {
	return di.FxProvideAs[Provider](provider, nil, append(anns, fx.ResultTags(`group:"rest_healthcheck_providers"`)))
}

var FxExport = fx.Options(
	ProvideAsHealthCheckProvider(NewDatabaseBunHealthCheckProviderFx),
	ProvideAsHealthCheckProvider(NewDatabaseGORMHealthCheckProviderFx),
	ProvideAsHealthCheckProvider(NewRedisHealthCheckProviderFx),
)
