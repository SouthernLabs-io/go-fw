package core

import (
	"reflect"

	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/errors"
)

type BaseParams struct {
	fx.In
	LF           *LoggerFactory
	FxLifecycle  fx.Lifecycle
	FxShutdowner fx.Shutdowner

	Conf Config
	DB   Database `optional:"true"`
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
