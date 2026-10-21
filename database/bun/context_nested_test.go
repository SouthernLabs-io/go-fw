package databasebun

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	fw_context "github.com/southernlabs-io/go-fw/context"
)

type nestedContextValueKey struct{}
type beginHookValueKey struct{}

type beginContextHook struct{}

func (beginContextHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	if event.Query == "BEGIN" {
		return context.WithValue(ctx, beginHookValueKey{}, "query-hook-context")
	}
	return ctx
}

func (beginContextHook) AfterQuery(context.Context, *bun.QueryEvent) {}

func TestRunInTx_NestedTransactionPreservesCallerContextAndRestoresDB(t *testing.T) {
	sqlDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`SAVEPOINT .+`).WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec(`ROLLBACK TO SAVEPOINT .+`).WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectRollback()

	db := bun.NewDB(sqlDB, pgdialect.New()).WithQueryHook(beginContextHook{})
	requestCtx := fw_context.NewContextWithStore(context.Background())
	requestCtx = AddToCtx(requestCtx, db)

	nestedErr := errors.New("nested callback failed")
	deadline := time.Now().Add(time.Minute)
	err = RunInTx(requestCtx, func(outerCtx context.Context) error {
		// Top-level Bun callback hooks remain visible to framework callbacks.
		require.Equal(t, "query-hook-context", outerCtx.Value(beginHookValueKey{}))
		outerTx, ok := GetDBFromCtx(outerCtx).(Tx)
		require.True(t, ok)

		callerCtx, cancel := context.WithCancel(context.WithValue(outerCtx, nestedContextValueKey{}, "caller-value"))
		defer cancel()
		callerCtx, cancelDeadline := context.WithDeadline(callerCtx, deadline)
		defer cancelDeadline()

		var innerCallbackCtx context.Context
		innerErr := RunInTx(callerCtx, func(innerCtx context.Context) error {
			innerCallbackCtx = innerCtx
			require.Equal(t, "caller-value", innerCtx.Value(nestedContextValueKey{}))
			gotDeadline, ok := innerCtx.Deadline()
			require.True(t, ok)
			require.Equal(t, deadline, gotDeadline)
			require.NotNil(t, innerCtx.Done())
			require.NoError(t, innerCtx.Err())

			innerTx, ok := GetDBFromCtx(innerCtx).(Tx)
			require.True(t, ok)
			// The inner framework transaction is backed by Bun's savepoint transaction.
			require.Equal(t, outerTx, innerTx.parentDB)
			return nestedErr
		})
		require.ErrorIs(t, innerErr, nestedErr)
		cancel()
		require.ErrorIs(t, innerCallbackCtx.Err(), context.Canceled)

		// The framework's mutable context store must point back at the outer tx
		// after the nested callback, even though Bun supplied another context.
		require.Equal(t, outerTx, GetDBFromCtx(requestCtx))
		return nestedErr
	})
	require.ErrorIs(t, err, nestedErr)

	// The original HTTP context store is restored even when the transaction fails.
	require.Same(t, db, GetDBFromCtx(requestCtx))
	require.NoError(t, sqlMock.ExpectationsWereMet())
}
