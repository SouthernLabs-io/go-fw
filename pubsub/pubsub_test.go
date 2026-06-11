package pubsub_test

import (
	"context"
	"database/sql"
	"io"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/southernlabs-io/go-fw/pubsub"
	"github.com/southernlabs-io/go-fw/pubsub/inmemory"
	"github.com/southernlabs-io/go-fw/pubsub/postgres"
	redisbroker "github.com/southernlabs-io/go-fw/pubsub/redis"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func setupPostgresDB(t *testing.T) *bun.DB {
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

func setupRedis(t *testing.T) redis.UniversalClient {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("Skipping redis tests: unable to connect to db: %v", err)
	}

	t.Cleanup(func() {
		_ = rdb.Close()
	})
	return rdb
}

func TestPubSubImplementations(t *testing.T) {
	backends := map[string]func(t *testing.T) pubsub.Broker{
		"InMemory": func(t *testing.T) pubsub.Broker {
			return inmemory.NewBroker()
		},
		"Postgres": func(t *testing.T) pubsub.Broker {
			db := setupPostgresDB(t)
			return postgres.NewBroker(db)
		},
		"Redis": func(t *testing.T) pubsub.Broker {
			rdb := setupRedis(t)
			return redisbroker.NewBroker(rdb)
		},
	}

	for name, factory := range backends {
		t.Run(name, func(t *testing.T) {
			t.Run("BasicPubSub", func(t *testing.T) { testBasicPubSub(t, factory) })
			t.Run("MultipleChannels", func(t *testing.T) { testMultipleChannels(t, factory) })
			t.Run("DynamicChannels", func(t *testing.T) { testDynamicChannels(t, factory) })
			t.Run("PublishAsync", func(t *testing.T) { testPublishAsync(t, factory) })
			t.Run("BrokerClosed", func(t *testing.T) { testBrokerClosed(t, factory) })
			t.Run("SubscriptionClosed", func(t *testing.T) { testSubscriptionClosed(t, factory) })
			t.Run("Drain", func(t *testing.T) { testDrain(t, factory) })
			t.Run("ContextCanceled", func(t *testing.T) { testContextCanceled(t, factory) })
		})
	}
}

func testBasicPubSub(t *testing.T, factory func(t *testing.T) pubsub.Broker) {
	b := factory(t)
	defer b.Close(context.Background())

	sub := b.Subscribe("test-ch")
	defer sub.Close(context.Background())

	time.Sleep(100 * time.Millisecond) // Allow network backends to LISTEN

	err := b.Publish(context.Background(), "test-ch", []byte("hello"))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	msg, err := sub.Recv(ctx)
	require.NoError(t, err)
	require.Equal(t, "test-ch", msg.Channel)
	require.Equal(t, []byte("hello"), msg.Payload)
}

func testMultipleChannels(t *testing.T, factory func(t *testing.T) pubsub.Broker) {
	b := factory(t)
	defer b.Close(context.Background())

	sub := b.Subscribe("ch1", "ch2")
	defer sub.Close(context.Background())

	time.Sleep(100 * time.Millisecond)

	err := b.Publish(context.Background(), "ch1", []byte("msg1"))
	require.NoError(t, err)
	err = b.Publish(context.Background(), "ch2", []byte("msg2"))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	msgs := make(map[string]string)
	for range 2 {
		msg, err := sub.Recv(ctx)
		require.NoError(t, err)
		msgs[msg.Channel] = string(msg.Payload)
	}

	require.Equal(t, "msg1", msgs["ch1"])
	require.Equal(t, "msg2", msgs["ch2"])
}

func testDynamicChannels(t *testing.T, factory func(t *testing.T) pubsub.Broker) {
	b := factory(t)
	defer b.Close(context.Background())

	sub := b.Subscribe()
	defer sub.Close(context.Background())

	err := sub.AddChannel("dyn-ch")
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = b.Publish(context.Background(), "dyn-ch", []byte("dyn1"))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	msg, err := sub.Recv(ctx)
	require.NoError(t, err)
	require.Equal(t, "dyn-ch", msg.Channel)
	require.Equal(t, []byte("dyn1"), msg.Payload)

	err = sub.RemoveChannel("dyn-ch")
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	err = b.Publish(context.Background(), "dyn-ch", []byte("dyn2"))
	require.NoError(t, err)

	ctx2, cancel2 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel2()
	_, err = sub.Recv(ctx2)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func testPublishAsync(t *testing.T, factory func(t *testing.T) pubsub.Broker) {
	b := factory(t)
	defer b.Close(context.Background())

	sub := b.Subscribe("async-ch")
	defer sub.Close(context.Background())

	time.Sleep(100 * time.Millisecond)

	err := b.PublishAsync(context.Background(), "async-ch", []byte("async-msg"))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	msg, err := sub.Recv(ctx)
	require.NoError(t, err)
	require.Equal(t, "async-ch", msg.Channel)
	require.Equal(t, []byte("async-msg"), msg.Payload)
}

func testBrokerClosed(t *testing.T, factory func(t *testing.T) pubsub.Broker) {
	b := factory(t)
	err := b.Close(context.Background())
	require.NoError(t, err)

	// Idempotent close
	err = b.Close(context.Background())
	require.NoError(t, err)

	sub := b.Subscribe("ch")

	// Should fail since broker is closed
	err = b.Publish(context.Background(), "ch", []byte("msg"))
	require.ErrorIs(t, err, pubsub.ErrBrokerClosed)

	err = b.PublishAsync(context.Background(), "ch", []byte("msg"))
	require.ErrorIs(t, err, pubsub.ErrBrokerClosed)

	_, err = sub.Recv(context.Background())
	require.ErrorIs(t, err, io.EOF)
}

func testSubscriptionClosed(t *testing.T, factory func(t *testing.T) pubsub.Broker) {
	b := factory(t)
	defer b.Close(context.Background())

	sub := b.Subscribe("close-ch")

	err := sub.Close(context.Background())
	require.NoError(t, err)

	// Idempotent close
	err = sub.Close(context.Background())
	require.NoError(t, err)

	err = sub.AddChannel("ch2")
	require.ErrorIs(t, err, io.EOF)

	err = sub.RemoveChannel("close-ch")
	require.ErrorIs(t, err, io.EOF)
}

func testDrain(t *testing.T, factory func(t *testing.T) pubsub.Broker) {
	b := factory(t)

	sub := b.Subscribe("drain-ch")

	time.Sleep(100 * time.Millisecond) // Allow network backends to LISTEN

	err := b.Publish(context.Background(), "drain-ch", []byte("msg"))
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond) // Ensure message reaches the local subscription buffer

	err = b.Close(context.Background())
	require.NoError(t, err)

	// Should still receive the message because it was buffered before close
	msg, err := sub.Recv(context.Background())
	require.NoError(t, err)
	require.Equal(t, []byte("msg"), msg.Payload)

	// Next one should be EOF
	_, err = sub.Recv(context.Background())
	require.ErrorIs(t, err, io.EOF)
}

func testContextCanceled(t *testing.T, factory func(t *testing.T) pubsub.Broker) {
	b := factory(t)
	defer b.Close(context.Background())

	sub := b.Subscribe("ch")
	defer sub.Close(context.Background())

	// Canceled context on Recv immediately returns context.Canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := sub.Recv(ctx)
	require.ErrorIs(t, err, context.Canceled)
}
