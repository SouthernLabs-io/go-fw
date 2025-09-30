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

// NewTx starts a new Bun transaction and returns a new context containing the transaction.
func NewTx(ctx context.Context) (context.Context, bun.Tx, error) {
	db := GetDBFromCtx(ctx)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return ctx, bun.Tx{}, err
	}

	ctx = AddToCtx(ctx, tx)
	return ctx, tx, nil
}
