package test

import (
	"context"

	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/database"
)

func NewContext(db database.DB, lf *core.LoggerFactory) context.Context {
	ctx := context.Background()
	ctx = lf.SetCtx(ctx)
	ctx = db.SetCtx(ctx)

	return ctx
}

var TestModuleContext = fx.Provide(NewContext)
