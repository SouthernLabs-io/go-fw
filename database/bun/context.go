package databasebun

import (
	"context"

	"github.com/uptrace/bun"

	fw_context "github.com/southernlabs-io/go-fw/context"
	"github.com/southernlabs-io/go-fw/database"
)

func GetDBFromCtx(ctx context.Context) bun.IDB {
	if idb, ok := ctx.Value(database.DBCtxKey).(bun.IDB); ok {
		return idb
	}
	return nil
}

func AddToCtx(ctx context.Context, db bun.IDB) context.Context {
	if db == nil {
		return ctx
	}
	return fw_context.WithValue(ctx, database.DBCtxKey, db)
}

func RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	idb := GetDBFromCtx(ctx)
	return idb.RunInTx(ctx, nil, func(ctxTx context.Context, tx bun.Tx) error {
		defer func() {
			// We need to restore the original DB in the context after the transaction ends if we are in an http request.
			// This is because the rest package uses a mutable context.
			if ctx == ctxTx {
				ctx = AddToCtx(ctx, idb)
			}
		}()
		ctxTx = AddToCtx(ctxTx, tx)
		return fn(ctxTx)
	})
}
