package test

import (
	"context"
	"testing"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/fxtest"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/redis"
)

type Target interface {
	GetBase() TargetBase
}

type TargetBase struct {
	fx.In

	Ctx  context.Context
	Conf core.Config

	DB    core.Database `optional:"true"`
	Redis redis.Redis   `optional:"true"`

	HTTPHandler core.HTTPHandler `optional:"true"`
}

func (f TargetBase) GetBase() TargetBase {
	return f
}

type FxApp struct {
	t    *testing.T
	opts fx.Option
	app  *fxtest.App
}

func FxUnit(t *testing.T, opts ...fx.Option) *FxApp {
	fxApp := &FxApp{
		t: t,
		opts: fx.Options(
			fx.Supply(t, fx.Annotate(t, fx.As(new(testing.TB)))),
			fx.Supply(NewConfig(t.Name())),
			fx.Provide(NewLoggerFactory),
			fx.Provide(ProvideCoreConfig),
			fx.WithLogger(func(lf *core.LoggerFactory) fxevent.Logger {
				return core.NewFxLogger(lf.GetLoggerForType(fx.App{}))
			}),
			TestModuleContext,
			fx.Options(opts...),
		),
	}

	return fxApp
}

func FxIntegration(t *testing.T, opts ...fx.Option) *FxApp {
	IntegrationTest(t)
	return FxUnit(
		t,
		fx.Options(opts...),
	)
}

// FxIntegrationWithDB is a helper function to create an integration test with DB
func FxIntegrationWithDB(t *testing.T, opts ...fx.Option) *FxApp {
	return FxIntegration(
		t,
		fx.Options(opts...),
		TestModuleDB,
	)
}

func (a *FxApp) WithDB() *FxApp {
	a.opts = fx.Options(a.opts, TestModuleDB)
	return a
}

func (a *FxApp) WithRedis() *FxApp {
	a.opts = fx.Options(a.opts, TestModuleRedis)
	return a
}

func (a *FxApp) WithHTTPHandler() *FxApp {
	a.opts = fx.Options(
		a.opts,
		TestModuleHTTPHandler,
		TestModuleMiddlewares,
		TestModuleRest,
	)
	return a
}

func (a *FxApp) WithWorkerHandler() *FxApp {
	a.opts = fx.Options(a.opts, TestModuleWorkerHandler)
	return a
}

func (a *FxApp) Populate(targets ...any) *FxApp {
	a.app = fxtest.New(
		a.t,
		a.opts,
		fx.Populate(targets...),
	).RequireStart()
	a.t.Cleanup(a.Stop)
	return a
}

func (a *FxApp) Stop() {
	a.app.RequireStop()
}

func (a *FxApp) Wait() <-chan fx.ShutdownSignal {
	return a.app.Wait()
}
