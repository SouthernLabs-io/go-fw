package databasebun

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"

	fw_context "github.com/southernlabs-io/go-fw/context"
	"github.com/southernlabs-io/go-fw/database"
)

// MockIDB is a mock implementation of bun.IDB for testing
type MockIDB struct {
	mock.Mock
}

func (m *MockIDB) RunInTx(ctx context.Context, opts *sql.TxOptions, fn func(context.Context, bun.Tx) error) error {
	args := m.Called(ctx, opts, fn)
	return args.Error(0)
}

func (m *MockIDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (bun.Tx, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return bun.Tx{}, args.Error(1)
	}
	return args.Get(0).(bun.Tx), args.Error(1)
}

// Implement IConn methods
func (m *MockIDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	callArgs := m.Called(ctx, query, args)
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).(*sql.Rows), callArgs.Error(1)
}

func (m *MockIDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	callArgs := m.Called(ctx, query, args)
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).(sql.Result), callArgs.Error(1)
}

func (m *MockIDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return m.Called(ctx, query, args).Get(0).(*sql.Row)
}

// Stub methods that satisfy bun.IDB interface
func (m *MockIDB) Dialect() schema.Dialect {
	return nil
}

func (m *MockIDB) NewValues(model interface{}) *bun.ValuesQuery {
	return nil
}

func (m *MockIDB) NewSelect() *bun.SelectQuery {
	return nil
}

func (m *MockIDB) NewInsert() *bun.InsertQuery {
	return nil
}

func (m *MockIDB) NewUpdate() *bun.UpdateQuery {
	return nil
}

func (m *MockIDB) NewDelete() *bun.DeleteQuery {
	return nil
}

func (m *MockIDB) NewMerge() *bun.MergeQuery {
	return nil
}

func (m *MockIDB) NewRaw(query string, args ...interface{}) *bun.RawQuery {
	return nil
}

func (m *MockIDB) NewCreateTable() *bun.CreateTableQuery {
	return nil
}

func (m *MockIDB) NewDropTable() *bun.DropTableQuery {
	return nil
}

func (m *MockIDB) NewCreateIndex() *bun.CreateIndexQuery {
	return nil
}

func (m *MockIDB) NewDropIndex() *bun.DropIndexQuery {
	return nil
}

func (m *MockIDB) NewTruncateTable() *bun.TruncateTableQuery {
	return nil
}

func (m *MockIDB) NewAddColumn() *bun.AddColumnQuery {
	return nil
}

func (m *MockIDB) NewDropColumn() *bun.DropColumnQuery {
	return nil
}

func TestGetDBFromCtx_WithDefaultKey(t *testing.T) {
	mockDB := &MockIDB{}
	ctx := fw_context.WithValue(context.Background(), database.DBCtxKey, mockDB)

	result := GetDBFromCtx(ctx)

	assert.Equal(t, mockDB, result)
}

func TestGetDBFromCtx_WithCustomKey(t *testing.T) {
	mockDB := &MockIDB{}
	customKey := fw_context.CtxKey("custom_db_key")
	ctx := fw_context.WithValue(context.Background(), database.DBCtxCustomKey, customKey)
	ctx = fw_context.WithValue(ctx, customKey, mockDB)

	result := GetDBFromCtx(ctx)

	assert.Equal(t, mockDB, result)
}

func TestGetDBFromCtx_NoDBInContext(t *testing.T) {
	ctx := context.Background()

	result := GetDBFromCtx(ctx)

	assert.Nil(t, result)
}

func TestGetDBFromCtx_InvalidTypeInContext(t *testing.T) {
	ctx := fw_context.WithValue(context.Background(), database.DBCtxKey, "not a db")

	result := GetDBFromCtx(ctx)

	assert.Nil(t, result)
}

func TestGetDBFromCtxLane_WithLane(t *testing.T) {
	mockDB := &MockIDB{}
	ctxKey := database.CreateDBCtxKey("test-lane")
	ctx := fw_context.WithValue(context.Background(), ctxKey, mockDB)

	result := GetDBFromCtxLane(ctx, "test-lane")

	assert.Equal(t, mockDB, result)
}

func TestGetDBFromCtxLane_EmptyLane(t *testing.T) {
	mockDB := &MockIDB{}
	ctx := fw_context.WithValue(context.Background(), database.DBCtxKey, mockDB)

	result := GetDBFromCtxLane(ctx, "")

	assert.Equal(t, mockDB, result)
}

func TestGetDBFromCtxLane_NoDBForLane(t *testing.T) {
	ctx := context.Background()

	result := GetDBFromCtxLane(ctx, "nonexistent-lane")

	assert.Nil(t, result)
}

func TestAddToCtx_WithValidDB(t *testing.T) {
	mockDB := &MockIDB{}
	ctx := context.Background()

	result := AddToCtx(ctx, mockDB)

	retrieved := GetDBFromCtx(result)
	assert.Equal(t, mockDB, retrieved)
}

func TestAddToCtx_WithNilDB(t *testing.T) {
	ctx := context.Background()

	result := AddToCtx(ctx, nil)

	assert.Equal(t, ctx, result)
}

func TestWithLane_EmptyLane(t *testing.T) {
	mockDB := &MockIDB{}
	ctx := AddToCtx(context.Background(), mockDB)

	result := WithLane(ctx, "")

	assert.Equal(t, ctx, result)
}

func TestWithLane_NoDBInContext(t *testing.T) {
	ctx := context.Background()

	result := WithLane(ctx, "test-lane")

	assert.Equal(t, ctx, result)
}

func TestWithLane_WithValidDB(t *testing.T) {
	mockDB := &MockIDB{}
	ctx := AddToCtx(context.Background(), mockDB)

	result := WithLane(ctx, "test-lane")

	// Verify that the lane-specific DB can be retrieved
	retrieved := GetDBFromCtxLane(result, "test-lane")
	assert.Equal(t, mockDB, retrieved)

	// Verify that GetDBFromCtx returns the lane-specific DB due to custom key being set
	retrieved2 := GetDBFromCtx(result)
	assert.Equal(t, mockDB, retrieved2)
}

func TestWithLane_ExistingLane(t *testing.T) {
	mockDB1 := &MockIDB{}
	mockDB2 := &MockIDB{}
	ctx := AddToCtx(context.Background(), mockDB1)
	ctx = AddToCtxWithLane(ctx, mockDB2, "test-lane")

	result := WithLane(ctx, "test-lane")

	// Should return the same context or context with the lane already set
	retrieved := GetDBFromCtxLane(result, "test-lane")
	assert.Equal(t, mockDB2, retrieved)
}

func TestAddToCtxWithLane_WithValidDB(t *testing.T) {
	mockDB := &MockIDB{}
	ctx := context.Background()

	result := AddToCtxWithLane(ctx, mockDB, "test-lane")

	retrieved := GetDBFromCtxLane(result, "test-lane")
	assert.Equal(t, mockDB, retrieved)

	// Verify that the custom key is set
	retrievedDefault := GetDBFromCtx(result)
	assert.Equal(t, mockDB, retrievedDefault)
}

func TestAddToCtxWithLane_WithEmptyLane(t *testing.T) {
	mockDB := &MockIDB{}
	ctx := context.Background()

	result := AddToCtxWithLane(ctx, mockDB, "")

	// Should add with default key
	retrieved := GetDBFromCtx(result)
	assert.Equal(t, mockDB, retrieved)
}

func TestAddToCtxWithLane_WithNilDB(t *testing.T) {
	ctx := context.Background()

	result := AddToCtxWithLane(ctx, nil, "test-lane")

	assert.Equal(t, ctx, result)
}

func TestRunInTx_WithValidDB(t *testing.T) {
	mockDB := &MockIDB{}
	ctx := AddToCtx(context.Background(), mockDB)
	funcCalled := false

	// Setup mock to capture the function and call it
	mockDB.On("RunInTx", ctx, (*sql.TxOptions)(nil), mock.MatchedBy(func(fn func(context.Context, bun.Tx) error) bool {
		return fn != nil
	})).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(context.Context, bun.Tx) error)
		txCtx := context.Background()
		err := fn(txCtx, bun.Tx{})
		require.NoError(t, err)
		funcCalled = true
	}).Return(nil)

	err := RunInTx(ctx, func(ctxTx context.Context) error {
		return nil
	})

	require.NoError(t, err)
	assert.True(t, funcCalled)
	mockDB.AssertExpectations(t)
}

func TestRunInTx_NoDBInContext(t *testing.T) {
	ctx := context.Background()

	err := RunInTx(ctx, func(ctxTx context.Context) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no database found in context")
}

func TestRunInTx_FunctionReturnsError(t *testing.T) {
	mockDB := &MockIDB{}
	ctx := AddToCtx(context.Background(), mockDB)
	expectedErr := assert.AnError

	// Setup mock to propagate errors
	mockDB.On("RunInTx", ctx, (*sql.TxOptions)(nil), mock.MatchedBy(func(fn func(context.Context, bun.Tx) error) bool {
		return fn != nil
	})).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(context.Context, bun.Tx) error)
		txCtx := context.Background()
		err := fn(txCtx, bun.Tx{})
		assert.Error(t, err)
	}).Return(expectedErr)

	err := RunInTx(ctx, func(ctxTx context.Context) error {
		return expectedErr
	})

	assert.Equal(t, expectedErr, err)
	mockDB.AssertExpectations(t)
}

func TestRunInTxWithOpts_WithValidDB(t *testing.T) {
	mockDB := &MockIDB{}
	opts := &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	}
	ctx := AddToCtx(context.Background(), mockDB)
	funcCalled := false

	// Setup mock to call the function
	mockDB.On("RunInTx", ctx, opts, mock.MatchedBy(func(fn func(context.Context, bun.Tx) error) bool {
		return fn != nil
	})).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(context.Context, bun.Tx) error)
		txCtx := context.Background()
		err := fn(txCtx, bun.Tx{})
		require.NoError(t, err)
		funcCalled = true
	}).Return(nil)

	err := RunInTxWithOpts(ctx, opts, func(ctxTx context.Context) error {
		return nil
	})

	require.NoError(t, err)
	assert.True(t, funcCalled)
	mockDB.AssertExpectations(t)
}

func TestRunInTxWithOpts_NoDBInContext(t *testing.T) {
	ctx := context.Background()

	err := RunInTxWithOpts(ctx, nil, func(ctxTx context.Context) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no database found in context")
}

func TestRunInTxWithOpts_WithNilOpts(t *testing.T) {
	mockDB := &MockIDB{}
	ctx := AddToCtx(context.Background(), mockDB)
	funcCalled := false

	// Setup mock with nil opts
	mockDB.On("RunInTx", ctx, (*sql.TxOptions)(nil), mock.MatchedBy(func(fn func(context.Context, bun.Tx) error) bool {
		return fn != nil
	})).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(context.Context, bun.Tx) error)
		txCtx := context.Background()
		err := fn(txCtx, bun.Tx{})
		require.NoError(t, err)
		funcCalled = true
	}).Return(nil)

	err := RunInTxWithOpts(ctx, nil, func(ctxTx context.Context) error {
		return nil
	})

	require.NoError(t, err)
	assert.True(t, funcCalled)
	mockDB.AssertExpectations(t)
}

func TestTxStruct(t *testing.T) {
	mockDB := &MockIDB{}
	// bun.Tx is a struct, not *sql.Tx
	mockBunTx := bun.Tx{}

	tx := Tx{
		Tx:       mockBunTx,
		parentDB: mockDB,
	}

	assert.Equal(t, mockDB, tx.parentDB)
	assert.Equal(t, mockBunTx, tx.Tx)
}

func TestAddToCtx_MultipleDBs(t *testing.T) {
	mockDB1 := &MockIDB{}
	mockDB2 := &MockIDB{}
	ctx := context.Background()

	// Add first DB
	ctx = AddToCtx(ctx, mockDB1)
	retrieved1 := GetDBFromCtx(ctx)
	assert.Equal(t, mockDB1, retrieved1)

	// Add second DB (should override)
	ctx = AddToCtx(ctx, mockDB2)
	retrieved2 := GetDBFromCtx(ctx)
	assert.Equal(t, mockDB2, retrieved2)
}

func TestWithLane_MultipleLanes(t *testing.T) {
	mockDB := &MockIDB{}
	ctx := AddToCtx(context.Background(), mockDB)

	// Add first lane
	ctx = WithLane(ctx, "lane1")
	lane1DB := GetDBFromCtxLane(ctx, "lane1")
	assert.Equal(t, mockDB, lane1DB)

	// Add second lane from original context with default DB
	baseCtx := AddToCtx(context.Background(), mockDB)
	ctx2 := WithLane(baseCtx, "lane2")
	lane2DB := GetDBFromCtxLane(ctx2, "lane2")
	assert.Equal(t, mockDB, lane2DB)
}
