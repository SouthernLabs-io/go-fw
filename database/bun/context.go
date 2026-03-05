package databasebun

import (
	"database/sql"

	"github.com/uptrace/bun"

	"github.com/southernlabs-io/go-fw/context"
	fw_context "github.com/southernlabs-io/go-fw/context"
	"github.com/southernlabs-io/go-fw/database"
	"github.com/southernlabs-io/go-fw/errors"
)

type Tx struct {
	bun.Tx
	parentDB bun.IDB
}

func GetDBFromCtx(ctx context.Context) bun.IDB {
	var ctxKey context.CtxKey
	if customKey, ok := ctx.Value(database.DBCtxCustomKey).(context.CtxKey); ok {
		ctxKey = customKey
	} else {
		ctxKey = database.DBCtxKey
	}

	if idb, ok := ctx.Value(ctxKey).(bun.IDB); ok {
		return idb
	}
	return nil
}

func GetDBFromCtxLane(ctx context.Context, lane string) bun.IDB {
	var ctxKey context.CtxKey
	if lane != "" {
		ctxKey = database.CreateDBCtxKey(lane)
	} else {
		ctxKey = database.DBCtxKey
	}

	if idb, ok := ctx.Value(ctxKey).(bun.IDB); ok {
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

// WithLane adds a lane-specific DB to the context. It retrieves the current DB from the context and adds a new entry with the lane name as part of the key. This allows for multiple lanes to have their own DB instances in the same context, which can be useful for handling different database connections or transactions.
// Calls to GetDBFromCtx on the returned context will use the lane specific DB instead of the default one.
// If name is empty, then the original context is returned.
func WithLane(ctx context.Context, lane string) context.Context {
	if lane == "" {
		return ctx
	}

	// Check if we already have this lane in the context
	if idb := GetDBFromCtxLane(ctx, lane); idb != nil {
		customKey := database.CreateDBCtxKey(lane)
		if currCustomKey, ok := ctx.Value(database.DBCtxCustomKey).(database.DBCtxCustomKeyType); ok && currCustomKey == customKey {
			return ctx
		}
		// Set the current lane custom key in the context so that GetDBFromCtx can use it to retrieve the correct DB instance for this lane.
		return fw_context.WithValue(ctx, database.DBCtxCustomKey, customKey)
	}

	// Get current IDB from context
	idb := GetDBFromCtx(ctx)
	if idb == nil {
		return ctx
	}

	// Lanes must start with *bun.DB, so we need to get the parent DB if we are in a transaction.
	if tx, ok := idb.(Tx); ok {
		idb = tx.parentDB
	}

	return AddToCtxWithLane(ctx, idb, lane)
}

func AddToCtxWithLane(ctx context.Context, db bun.IDB, name string) context.Context {
	if db == nil {
		return ctx
	}

	if name == "" {
		return AddToCtx(ctx, db)
	}

	ctxKey := database.CreateDBCtxKey(name)
	ctx = fw_context.WithValue(ctx, ctxKey, db)
	// Store the custom key in the context so that we can retrieve it later in GetDBFromCtx.
	ctx = fw_context.WithValue(ctx, database.DBCtxCustomKey, ctxKey)
	return ctx
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
		ctxTx = AddToCtx(ctxTx, Tx{Tx: tx, parentDB: idb})
		return fn(ctxTx)
	})
}

func RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return RunInTxWithOpts(ctx, nil, fn)
}
