package test

import (
	"context"

	"go.uber.org/fx"

	databasebun "github.com/southernlabs-io/go-fw/database/bun"
	databasegorm "github.com/southernlabs-io/go-fw/database/gorm"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/uptrace/bun"
)

func NewContext(dbGORM databasegorm.DB, dbBun *bun.DB, lf log.LoggerFactory) context.Context {
	ctx := context.Background()
	ctx = lf.AddToCtx(ctx)
	ctx = dbGORM.AddToCtx(ctx)
	if dbBun != nil {
		ctx = databasebun.AddToCtx(ctx, dbBun)
	}

	return ctx
}

var FxExportContext = fx.Provide(fx.Annotate(NewContext, fx.ParamTags(`optional:"true"`, `optional:"true"`)))
