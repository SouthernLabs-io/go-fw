package database_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/database"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/test"
)

func TestDBTx(t *testing.T) {
	test.IntegrationTest(t)

	conf := test.NewTestConfig(test.NewTestRootConfig(t))
	lf := test.NewLoggerFactory(t, conf.RootConfig)
	conf.Database = config.DatabaseConfig{
		Host: "localhost",
		Port: 5432,
		User: "postgres",
		Pass: "postgres",
	}
	db := test.NewTestDatabase(conf, lf)
	defer func(conf config.Config, lf *log.LoggerFactory, db database.DB) {
		err := test.OnTestDBStop(conf, db, lf)
		require.NoError(t, err)
	}(conf, lf, db)
	ctx := test.NewContext(db, lf)

	tx, ctx2 := database.WithTx(ctx)
	require.NotNil(t, tx)
	require.NotNil(t, ctx2)
	require.False(t, tx.IsAutomatic())
	require.False(t, tx.IsClosed())

	creatTableSQL := "CREATE TABLE test (id text not null)"
	subTxCount := 3
	for i := 0; i < subTxCount; i++ {
		subTx, ctx3 := database.WithTx(ctx2)
		require.NotNil(t, subTx)
		require.NotNil(t, ctx3)
		require.False(t, subTx.IsAutomatic())
		require.False(t, subTx.IsClosed())
		err := subTx.Raw(creatTableSQL).Row().Err()
		require.Nil(t, err)

		// Commit on the last iteration
		if i+1 == subTxCount {
			err = subTx.Commit().Error
		} else {
			err = subTx.Rollback().Error
		}

		require.Nil(t, err)
		require.True(t, subTx.IsClosed())
		require.Panics(t, func() {
			subTx.Exec(";")
		})
		require.ErrorIs(t, subTx.Error, sql.ErrTxDone)
	}

	err := tx.Raw("INSERT INTO test VALUES('123')").Row().Err()
	require.Nil(t, err)

	err = tx.Commit().Error
	require.Nil(t, err)
	require.True(t, tx.IsClosed())
	require.Panics(t, func() {
		tx.Exec(";")
	})
	require.ErrorIs(t, tx.Error, sql.ErrTxDone)

	var count int64
	err = database.InTx(ctx2).Raw("SELECT COUNT(*) FROM test").Row().Scan(&count)
	require.Nil(t, err)
	require.EqualValues(t, 1, count)
}
