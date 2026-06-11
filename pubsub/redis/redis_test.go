package redis

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/southernlabs-io/go-fw/pubsub"
	"github.com/stretchr/testify/require"
)

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

func TestRedisBroker_DBClosedUnderneath(t *testing.T) {
	rdb := setupRedis(t)
	b := NewBroker(rdb)
	defer b.Close(context.Background())

	sub := b.Subscribe("ch1")
	defer sub.Close(context.Background())

	time.Sleep(100 * time.Millisecond)

	// Close the DB underneath
	err := rdb.Close()
	require.NoError(t, err)

	// Publish should now fail with a network-level error
	err = b.Publish(context.Background(), "ch1", []byte("msg"))
	require.Error(t, err)
	require.NotErrorIs(t, err, pubsub.ErrBrokerClosed) // It should be a real network/db error
}

func TestRedisBroker_IdempotentCloseError(t *testing.T) {
	rdb := setupRedis(t)
	b := NewBroker(rdb)

	_ = rdb.Close()

	err1 := b.Close(context.Background())
	err2 := b.Close(context.Background())

	if err1 != nil {
		require.Equal(t, err1.Error(), err2.Error())
	} else {
		require.Equal(t, err1, err2)
	}
}

func TestRedisBroker_SilentSubscriptionFailures(t *testing.T) {
	rdb := setupRedis(t)
	b := NewBroker(rdb)
	defer b.Close(context.Background())

	sub := b.Subscribe()
	defer sub.Close(context.Background())

	redisBroker := b.(*broker)
	_ = redisBroker.pubsub.Close()

	err := sub.AddChannel("fail-ch")
	require.Error(t, err)

	redisBroker.mu.RLock()
	_, exists := redisBroker.subsByChannel["fail-ch"]
	redisBroker.mu.RUnlock()
	require.False(t, exists, "failed channel should be cleaned up from subsByChannel")
}
