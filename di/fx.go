package di

import (
	"reflect"

	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/database"
	"github.com/southernlabs-io/go-fw/errors"
)

type BaseParams struct {
	fx.In
	LF           *core.LoggerFactory
	FxLifecycle  fx.Lifecycle
	FxShutdowner fx.Shutdowner

	Conf core.Config
	DB   database.DB `optional:"true"`
}

func FxProvideAs[I any](provider any, tAnns []fx.Annotation, iAnns []fx.Annotation) fx.Option {
	providerT := reflect.TypeOf(provider)
	if providerT.Kind() != reflect.Func {
		panic(errors.Newf(errors.ErrCodeBadArgument, "provider must be a function, got: %T", provider))
	}

	return fx.Options(
		// Provide Type implementation
		fx.Provide(
			fx.Annotate(
				provider,
				tAnns...,
			),
		),
		// Provide as Interface
		fx.Provide(
			fx.Annotate(
				func(provided I) I {
					return provided
				},
				append(iAnns, fx.From(reflect.New(reflect.TypeOf(provider).Out(0)).Interface()))...,
			),
		),
	)
}
