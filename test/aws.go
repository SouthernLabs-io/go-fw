package test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"go.uber.org/fx"
)

func NewLocalStackAWSConfig(tb testing.TB) (aws.Config, error) {
	tb.Setenv("AWS_REGION", "us-east-1")
	tb.Setenv("AWS_ENDPOINT_URL", "http://localhost:4566")
	tb.Setenv("AWS_MAX_ATTEMPTS", "10")
	tb.Setenv("AWS_ACCESS_KEY_ID", "localstack")
	tb.Setenv("AWS_SECRET_ACCESS_KEY", "localstack")

	return awsconfig.LoadDefaultConfig(context.Background())
}

var ModuleTestAWSLocalStackConfig = fx.Options(
	fx.Provide(NewLocalStackAWSConfig),
)
