package secrets

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/southernlabs-io/go-fw/config"
	"go.uber.org/fx"
)

// EnvSecretsManager implements the SecretsManager interface using environment variables.
type EnvSecretsManager struct {
	keyTransformer KeyTransformer
}

var _ SecretsManager = (*EnvSecretsManager)(nil)

func NewEnvSecretsManagerFx(deps struct {
	fx.In

	RootConf       config.RootConfig
	KeyTransformer KeyTransformer `optional:"true"`
}) *EnvSecretsManager {
	if deps.KeyTransformer == nil {
		deps.KeyTransformer = NewDefaultKeyTransformer(deps.RootConf)
	}

	return &EnvSecretsManager{deps.KeyTransformer}
}

func (e *EnvSecretsManager) GetSecret(ctx context.Context, key string) (string, error) {
	id := e.keyTransformer.Transform(key)

	return e.GetSecretVerbatim(ctx, id)
}

func (e *EnvSecretsManager) GetSecretVerbatim(ctx context.Context, id string) (string, error) {
	value, ok := os.LookupEnv(id)
	if !ok {
		// Try upper case
		value, ok = os.LookupEnv(strings.ToUpper(id))
	}
	if !ok {
		return "", fmt.Errorf("environment variable %s not set", id)
	}
	return value, nil
}

func (e *EnvSecretsManager) GetBinarySecret(ctx context.Context, key string) ([]byte, error) {
	return e.GetBinarySecretVerbatim(ctx, key)
}

func (e *EnvSecretsManager) GetBinarySecretVerbatim(ctx context.Context, id string) ([]byte, error) {
	value, err := e.GetSecretVerbatim(ctx, id)
	if err != nil {
		return nil, err
	}

	return []byte(value), nil
}

var FxExportEnvSM = fx.Options(
	fxExport,
	ProvideAsSecretsManager(NewEnvSecretsManagerFx),
)
