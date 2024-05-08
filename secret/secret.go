package secret

import (
	"context"

	"go.uber.org/fx"
)

type SecretsManager interface {
	GetSecret(ctx context.Context, name string) (string, error)
	GetSecretVerbatim(ctx context.Context, id string) (string, error)
	GetBinarySecret(ctx context.Context, id string) ([]byte, error)
}

var Module = fx.Provide(NewAWSSecretsManager)
