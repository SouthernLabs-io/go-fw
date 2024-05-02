package test

import (
	"context"

	"go.uber.org/fx"

	lib "github.com/southernlabs-io/go-fw/core"
)

func NewContext(db lib.Database, lf *lib.LoggerFactory) context.Context {
	return lib.NewContextWithDeps(context.Background(), db, lf)
}

var TestModuleContext = fx.Provide(NewContext)
