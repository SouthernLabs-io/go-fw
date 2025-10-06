package secrets

import (
	"context"

	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/di"
)

type SecretsManager interface {
	// GetSecret retrieves a secret from the secret manager applying transformations to the provided key.
	GetSecret(ctx context.Context, key string) (string, error)

	// GetSecretVerbatim retrieves a secret from the secret manager using the provided id.
	// No transformations are done on the id.
	GetSecretVerbatim(ctx context.Context, id string) (string, error)

	// GetBinarySecret retrieves a binary secret from the secret manager applying transformations to the provided key.
	GetBinarySecret(ctx context.Context, key string) ([]byte, error)

	// GetBinarySecretVerbatim retrieves a binary secret from the secret manager using the provided id.
	// No transformations are done on the id.
	GetBinarySecretVerbatim(ctx context.Context, id string) ([]byte, error)
}

func ProvideAsSecretsManager(provider any, anns ...fx.Annotation) fx.Option {
	return di.FxProvideAs[SecretsManager](provider, nil, anns)
}

var fxExport = fx.Options(
	// Provide interface conversion for SecretsManager to config.SecretsManager
	fx.Provide(func(secretsManager SecretsManager) config.SecretsManager {
		return secretsManager
	}),
)
