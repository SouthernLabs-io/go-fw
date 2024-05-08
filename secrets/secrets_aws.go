package secrets

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/di"
	"github.com/southernlabs-io/go-fw/log"
)

type AWSSecretsManager struct {
	keyTransformer KeyTransformer
	client         *secretsmanager.Client
}

func NewAWSSecretsManager(deps struct {
	fx.In

	RootConf       config.RootConfig
	AwsConfig      *aws.Config
	KeyTransformer KeyTransformer `optional:"true"`
}) *AWSSecretsManager {
	if deps.KeyTransformer == nil {
		deps.KeyTransformer = NewDefaultKeyTransformer(deps.RootConf)
	}
	return &AWSSecretsManager{
		keyTransformer: deps.KeyTransformer,
		client:         secretsmanager.NewFromConfig(*deps.AwsConfig),
	}
}

func (s *AWSSecretsManager) GetSecret(ctx context.Context, key string) (string, error) {
	log.GetLoggerFromCtx(ctx).Infof("GetSecret: %s", key)
	fullId := s.keyTransformer.Transform(key)
	return s.GetSecretVerbatim(ctx, fullId)
}

func (s *AWSSecretsManager) GetSecretVerbatim(ctx context.Context, id string) (string, error) {
	log.GetLoggerFromCtx(ctx).Infof("GetSecretVerbatim: %s", id)
	resp, err := s.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &id,
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.SecretString), nil
}

func (s *AWSSecretsManager) GetBinarySecret(ctx context.Context, key string) ([]byte, error) {
	log.GetLoggerFromCtx(ctx).Infof("GetSecret: %s", key)
	fullId := s.keyTransformer.Transform(key)
	return s.GetBinarySecretVerbatim(ctx, fullId)
}

func (s *AWSSecretsManager) GetBinarySecretVerbatim(ctx context.Context, id string) ([]byte, error) {
	log.GetLoggerFromCtx(ctx).Infof("GetBinarySecretVerbatim: %s", id)
	resp, err := s.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &id,
	})
	if err != nil {
		return nil, err
	}
	return resp.SecretBinary, nil
}

var ModuleAWS = fx.Options(
	Module,
	di.FxProvideAs[SecretsManager](NewAWSSecretsManager, nil, nil),
)
