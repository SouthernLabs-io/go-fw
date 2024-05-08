package test

import (
	"context"

	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/database"
	"github.com/southernlabs-io/go-fw/log"
)

func NewContext(db database.DB, lf *log.LoggerFactory) context.Context {
	ctx := context.Background()
	ctx = lf.SetCtx(ctx)
	ctx = db.SetCtx(ctx)

	return ctx
}

var ModuleContext = fx.Provide(fx.Annotate(NewContext, fx.ParamTags(`optional:"true"`)))
