package secrets_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/secrets"
	"github.com/southernlabs-io/go-fw/test"
)

// Requires LocalStack (make docker_up); skipped under -short.
func TestAWSSecretsManager_JSONGroupedSecrets(t *testing.T) {
	var (
		ctx      context.Context
		rootConf config.RootConfig
		awsConf  aws.Config
		sm       secrets.SecretsManager
	)
	test.FxIntegration(t,
		test.FxExportTestAWSLocalStackConfig,
		fx.Provide(func(c aws.Config) *aws.Config { return &c }),
		secrets.FxExportAWSSM,
	).Populate(&ctx, &rootConf, &awsConf, &sm)

	client := secretsmanager.NewFromConfig(awsConf)
	createSecret := func(name, value string) {
		_, err := client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
			Name:         aws.String(name),
			SecretString: aws.String(value),
		})
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = client.DeleteSecret(context.Background(), &secretsmanager.DeleteSecretInput{
				SecretId:                   aws.String(name),
				ForceDeleteWithoutRecovery: aws.Bool(true),
			})
		})
	}

	// Transformed id: what <secret#> under Database resolves to with this RootConfig.
	groupID := secrets.NewDefaultKeyTransformer(rootConf).Transform("Database")
	createSecret(groupID, `{"User":"app","Pass":"s3cr3t","Port":6543}`)

	// Verbatim id with nested members.
	llmID := fmt.Sprintf("llm-keys-%d", time.Now().UnixNano())
	createSecret(llmID, `{"openai":{"apiKey":"sk-openai"},"anthropic":{"apiKey":"sk-ant"}}`)

	// A whole-secret reference to JSON content must come back untouched.
	wholeID := fmt.Sprintf("gcp-sa-%d", time.Now().UnixNano())
	createSecret(wholeID, `{"type":"service_account"}`)

	t.Setenv("DATABASE_USER", "<secret#>")
	t.Setenv("DATABASE_PASS", "<secret#>")
	t.Setenv("DATABASE_PORT", "<secret#>")
	t.Setenv("OPENAIKEY", "<secret:"+llmID+"#/openai/apiKey>")
	t.Setenv("ANTHROPICKEY", "<secret:"+llmID+"#/anthropic/apiKey>")
	t.Setenv("GCPSA", "<secret:"+wholeID+">")

	type Config struct {
		config.Config

		OpenAIKey    string
		AnthropicKey string
		GcpSA        string
	}
	var conf Config
	config.LoadConfig(rootConf, &conf, sm)

	require.Equal(t, "app", conf.Database.User)
	require.Equal(t, "s3cr3t", conf.Database.Pass)
	require.Equal(t, 6543, conf.Database.Port)
	require.Equal(t, "sk-openai", conf.OpenAIKey)
	require.Equal(t, "sk-ant", conf.AnthropicKey)
	require.Equal(t, `{"type":"service_account"}`, conf.GcpSA)
}
