package test

import (
	"context"

	"go.uber.org/fx"

	database "github.com/southernlabs-io/go-fw/database/gorm"
	"github.com/southernlabs-io/go-fw/log"
)

func NewContext(db database.DB, lf log.LoggerFactory) context.Context {
	ctx := context.Background()
	ctx = lf.AddToCtx(ctx)
	ctx = db.AddToCtx(ctx)

	return ctx
}

var FxExportContext = fx.Provide(fx.Annotate(NewContext, fx.ParamTags(`optional:"true"`)))
