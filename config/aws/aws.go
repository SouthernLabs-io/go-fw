package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"go.uber.org/fx"
	awstrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/aws/aws-sdk-go-v2/aws"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/errors"
)

func NewAWSConfig(rootConf config.RootConfig) *aws.Config {
	awsConfig, err := awsconfig.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(errors.NewUnknownf("failed to build default AWS config, error: %w", err))
	}

	if rootConf.Datadog.Tracing {
		awstrace.AppendMiddleware(&awsConfig)
	}
	return &awsConfig
}

var Module = fx.Provide(NewAWSConfig)
