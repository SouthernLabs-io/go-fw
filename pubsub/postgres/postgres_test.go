package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/southernlabs-io/go-fw/pubsub"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func setupDB(t *testing.T) *bun.DB {
	dsn := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := sqldb.PingContext(ctx); err != nil {
		t.Skipf("Skipping postgres tests: unable to connect to db: %v", err)
	}

	db := bun.NewDB(sqldb, pgdialect.New())
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func TestPostgresBroker_DBClosedUnderneath(t *testing.T) {
	db := setupDB(t)
	b := NewBroker(db)
	defer b.Close(context.Background())

	sub := b.Subscribe("ch1")
	defer sub.Close(context.Background())

	time.Sleep(100 * time.Millisecond)

	// Close the DB underneath
	err := db.Close()
	require.NoError(t, err)

	// Publish should now fail with a DB-level error
	err = b.Publish(context.Background(), "ch1", []byte("msg"))
	require.Error(t, err)
	require.NotErrorIs(t, err, pubsub.ErrBrokerClosed) // It should be a real network/db error
}

func TestPostgresBroker_IdempotentCloseError(t *testing.T) {
	db := setupDB(t)
	b := NewBroker(db)

	_ = db.Close()

	err1 := b.Close(context.Background())
	err2 := b.Close(context.Background())

	if err1 != nil {
		require.Equal(t, err1.Error(), err2.Error())
	} else {
		require.Equal(t, err1, err2)
	}
}

func TestPostgresBroker_SilentSubscriptionFailures(t *testing.T) {
	db := setupDB(t)
	b := NewBroker(db)
	defer b.Close(context.Background())

	sub := b.Subscribe()
	defer sub.Close(context.Background())

	pgBroker := b.(*broker)
	_ = pgBroker.ln.Close()

	err := sub.AddChannel("fail-ch")
	require.Error(t, err)

	pgBroker.mu.RLock()
	_, exists := pgBroker.subsByChannel["fail-ch"]
	pgBroker.mu.RUnlock()
	require.False(t, exists, "failed channel should be cleaned up from subsByChannel")
}
