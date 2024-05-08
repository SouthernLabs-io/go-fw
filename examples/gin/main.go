package main

import (
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/bootstrap"
	"github.com/southernlabs-io/go-fw/config/aws"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/panics"
	"github.com/southernlabs-io/go-fw/rest"
	"github.com/southernlabs-io/go-fw/rest/middleware"
	"github.com/southernlabs-io/go-fw/secrets"
	"github.com/southernlabs-io/go-fw/slack"
)

func main() {
	defer panics.DeferredPanicToLogAndExit()
	var deps = fx.Options(

		// Service configuration
		fx.Provide(NewConfig),

		// AWS Config
		fx.Provide(aws.NewAWSConfig),

		// Secrets with AWS
		secrets.ModuleAWS,

		// Middlewares
		middleware.RequestLoggerModule,

		// Use Slack notifications
		slack.Module,

		rest.ProvideAsResource(NewMyResource),
	)
	err := bootstrap.NewAppWithServe(deps).Execute()
	if err != nil {
		panic(errors.NewUnknownf("failed to execute with error: %w", err))
	}
}
