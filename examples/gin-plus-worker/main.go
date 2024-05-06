package main

import (
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/bootstrap"
	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/rest"
	"github.com/southernlabs-io/go-fw/rest/middlewares"
	"github.com/southernlabs-io/go-fw/worker"
)

func main() {
	defer core.DeferredPanicToLogAndExit()
	var commonDeps = fx.Options(
		// middlewares
		middlewares.RequestLoggerModule,
	)

	var serverDeps = fx.Options(
		rest.ProvideAsResource(NewSimpleResource),
	)

	var workerDeps = fx.Options(
		worker.ProvideAsLongRunningWorker(NewSimpleWorker),
	)
	err := bootstrap.NewAppWithServeAndWork(commonDeps, serverDeps, workerDeps).Execute()
	if err != nil {
		panic(errors.NewUnknownf("failed to execute with error: %w", err))
	}
}
