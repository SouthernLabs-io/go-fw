package databasebun

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"

	fw_context "github.com/southernlabs-io/go-fw/context"
	"github.com/southernlabs-io/go-fw/database"
	"github.com/southernlabs-io/go-fw/errors"
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

func RunInTxWithOpts(ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context) error) error {
	idb := GetDBFromCtx(ctx)
	if idb == nil {
		return errors.NewUnknownf("no database found in context")
	}
	return idb.RunInTx(ctx, opts, func(ctxTx context.Context, tx bun.Tx) error {
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

func RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return RunInTxWithOpts(ctx, nil, fn)
}
