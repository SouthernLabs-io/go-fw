package di

import (
	"reflect"

	"github.com/uptrace/bun"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	databasegorm "github.com/southernlabs-io/go-fw/database/gorm"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
)

type BaseParams struct {
	fx.In
	LF           log.LoggerFactory
	FxLifecycle  fx.Lifecycle
	FxShutdowner fx.Shutdowner

	Conf   config.Config
	DBGorm databasegorm.DB `optional:"true"`
	DBBun  *bun.DB         `optional:"true"`
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

var FxExport = fx.Provide(NewFxLoggerFactory)
