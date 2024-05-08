package secret

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/log"
)

//NOTE: it is required to keep this implementation on the lib package to avoid
// recursive imports error.

// Example: awesome-service/dev1/
const defaultPrefixFmt = "%s/%s"
const defaultKeyFmt = "%s"

type AWSSecretsManager struct {
	logger log.Logger
	prefix string
	keyFmt string
	client *secretsmanager.Client
}

func NewAWSSecretsManager(
	conf config.RootConfig,
	awsConfig *aws.Config,
	lf *log.LoggerFactory,
) SecretsManager {
	prefixFmt := conf.Secrets.PrefixFmt
	if prefixFmt == "" {
		prefixFmt = defaultPrefixFmt
	}

	keyFmt := conf.Secrets.KeyFmt
	if keyFmt == "" {
		keyFmt = defaultKeyFmt
	}

	return &AWSSecretsManager{
		logger: lf.GetLoggerForType(AWSSecretsManager{}),
		prefix: fmt.Sprintf(prefixFmt, conf.Name, conf.Env.Name),
		keyFmt: keyFmt,
		client: secretsmanager.NewFromConfig(*awsConfig),
	}
}

func (s *AWSSecretsManager) GetSecret(ctx context.Context, name string) (string, error) {
	fullId := s.prefix + "/" + fmt.Sprintf(s.keyFmt, name)
	return s.GetSecretVerbatim(ctx, fullId)
}

func (s *AWSSecretsManager) GetSecretVerbatim(ctx context.Context, id string) (string, error) {
	s.logger.Infof("GetSecret: %s", id)
	resp, err := s.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &id,
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.SecretString), nil
}

func (s *AWSSecretsManager) GetBinarySecret(ctx context.Context, secretId string) ([]byte, error) {
	fullKey := s.prefix + secretId
	s.logger.Infof("GetBinarySecret: %s", fullKey)
	resp, err := s.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &fullKey,
	})
	if err != nil {
		return nil, err
	}
	return resp.SecretBinary, nil
}
