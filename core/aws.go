package core

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awstrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/aws/aws-sdk-go-v2/aws"

	"github.com/southernlabs-io/go-fw/errors"
)

func NewAWSConfig(conf CoreConfig) *aws.Config {
	awsConfig, err := awsconfig.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(errors.NewUnknownf("failed to build default AWS config, error: %w", err))
	}

	if conf.Datadog.Tracing {
		awstrace.AppendMiddleware(&awsConfig)
	}
	return &awsConfig
}