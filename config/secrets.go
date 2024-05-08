package config

import (
	"context"

	"github.com/southernlabs-io/go-fw/errors"
)

// SecretsManager is a minimal interface needed to process config secrets
type SecretsManager interface {
	GetSecret(ctx context.Context, name string) (string, error)
	GetSecretVerbatim(ctx context.Context, id string) (string, error)
}

type PanicSecretsManager struct {
}

func (w PanicSecretsManager) GetSecret(_ context.Context, name string) (string, error) {
	panic(errors.Newf(errors.ErrCodeBadState, "no SecretsManager provided, but trying to get secret for key: %s", name))
}

func (w PanicSecretsManager) GetSecretVerbatim(_ context.Context, id string) (string, error) {
	panic(errors.Newf(errors.ErrCodeBadState, "no SecretsManager provided, but trying to get secret for key: %s", id))
}
