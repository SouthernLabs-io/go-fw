package config

import (
	"context"

	"github.com/southernlabs-io/go-fw/errors"
)

// SecretsManager is a minimal interface needed to process config secrets.
//
// Values are returned as opaque strings. When a config placeholder carries a '#' selector
// (<secret#/field>, <secret:ID#/field>), the loader parses the returned string as a JSON object and
// extracts the member addressed by the RFC 6901 JSON Pointer, so one secret can hold many related
// values. Implementations do not need to know about this.
type SecretsManager interface {
	GetSecret(ctx context.Context, key string) (string, error)
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
