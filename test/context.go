package test

import (
	"context"

	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/core"
)

func NewContext(db core.Database, lf *core.LoggerFactory) context.Context {
	return core.NewContextWithDeps(context.Background(), db, lf)
}

var TestModuleContext = fx.Provide(NewContext)
