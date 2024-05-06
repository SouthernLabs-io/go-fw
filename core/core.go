package core

import (
	"context"

	"go.uber.org/fx"
)

type SecretsManager interface {
	GetSecret(ctx context.Context, name string) (string, error)
	GetSecretVerbatim(ctx context.Context, id string) (string, error)
	GetBinarySecret(ctx context.Context, id string) ([]byte, error)
}

// Module exports dependency
var Module = fx.Options(
	fx.Provide(NewAWSConfig),
	fx.Provide(NewAWSSecretsManager),
	fx.Provide(NewConfig),
	fx.Provide(NewCoreConfig),
	fx.Provide(NewLoggerFactory),
	fx.Provide(NewSlackClient),
	fx.Provide(NewSlackFxLifecycleLoggerInterceptor),
)
