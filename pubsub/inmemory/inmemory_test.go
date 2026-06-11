package inmemory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// This test is specific to the inmemory broker because it relies on the
// exact backpressure mechanism (a channel of size 100). Networked brokers
// handle backpressure differently or not at all.
func TestInMemoryBroker_Backpressure(t *testing.T) {
	b := NewBroker()
	defer b.Close(context.Background())

	sub := b.Subscribe("ch")
	defer sub.Close(context.Background())

	// Send 100 messages to fill the buffer
	for i := 0; i < 100; i++ {
		err := b.Publish(context.Background(), "ch", []byte("fill"))
		require.NoError(t, err)
	}

	// Now the 101st message should block. If we pass a canceled context, it should return context.Canceled.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := b.Publish(ctx, "ch", []byte("blocked"))
	require.ErrorIs(t, err, context.Canceled)
}
