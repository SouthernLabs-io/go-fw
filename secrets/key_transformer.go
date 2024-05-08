package secrets

import (
	"strings"

	"github.com/southernlabs-io/go-fw/config"
)

// KeyTransformer is an interface for transforming a secret key into a full secret ID
type KeyTransformer interface {
	Transform(name string) string
}

const defaultPrefixFmt = "{Name}/{Env.Type}/{Env.Name}/"
const defaultKeyFmt = "{Key}"

// DefaultKeyTransformer is a default implementation of KeyTransformer
// It use config.SecretsConfig to format the ID.
// The supported placeholders are:
// - {Name} - the name of the service
// - {Env.Name} - the name of the environment
// - {Env.Type} - the type of the environment
// - {Key} - the secret name
type DefaultKeyTransformer struct {
	prefix string
	keyFmt string
}

func NewDefaultKeyTransformer(conf config.RootConfig) KeyTransformer {
	prefixFmt := conf.Secrets.PrefixFmt
	if prefixFmt == "" {
		prefixFmt = defaultPrefixFmt
	}
	prefix := strings.ReplaceAll(prefixFmt, "{Name}", conf.Name)
	prefix = strings.ReplaceAll(prefix, "{Env.Name}", conf.Env.Name)
	prefix = strings.ReplaceAll(prefix, "{Env.Type}", string(conf.Env.Type))

	keyFmt := conf.Secrets.KeyFmt
	if keyFmt == "" {
		keyFmt = defaultKeyFmt
	}

	keyFmt = strings.ReplaceAll(keyFmt, "{Name}", conf.Name)
	keyFmt = strings.ReplaceAll(keyFmt, "{Env.Name}", conf.Env.Name)
	keyFmt = strings.ReplaceAll(keyFmt, "{Env.Type}", string(conf.Env.Type))

	return &DefaultKeyTransformer{
		prefix: prefix,
		keyFmt: keyFmt,
	}
}

func (t *DefaultKeyTransformer) Transform(key string) string {
	return t.prefix + strings.ReplaceAll(t.keyFmt, "{Key}", key)
}
