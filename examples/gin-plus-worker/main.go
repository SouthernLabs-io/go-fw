package main

import (
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/bootstrap"
	"github.com/southernlabs-io/go-fw/distributedlock"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/panics"
	"github.com/southernlabs-io/go-fw/rest"
	"github.com/southernlabs-io/go-fw/rest/middleware"
	"github.com/southernlabs-io/go-fw/worker"
)

func main() {
	defer panics.DeferredPanicToLogAndExit()
	var commonDeps = fx.Options(
		// middlewares
		middleware.RequestLoggerModule,
	)

	var serverDeps = fx.Options(
		rest.ProvideAsResource(NewSimpleResource),
	)

	var workerDeps = fx.Options(
		distributedlock.ModuleLocal,
		worker.ProvideAsLongRunningWorker(NewSimpleWorker),
	)
	err := bootstrap.NewAppWithServeAndWork(commonDeps, serverDeps, workerDeps).Execute()
	if err != nil {
		panic(errors.NewUnknownf("failed to execute with error: %w", err))
	}
}
