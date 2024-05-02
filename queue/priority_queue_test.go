package queue_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/queue"
)

func TestPriorityQueue(t *testing.T) {
	pq := queue.NewPriorityQueue[string]()
	require.NotNil(t, pq)
	require.EqualValues(t, 0, pq.Len())

	pq.Push("last", 1_000)
	require.EqualValues(t, 1, pq.Len())

	pq.Push("first", 0)
	require.EqualValues(t, 2, pq.Len())

	midItem := pq.Push("mid", 500)
	require.EqualValues(t, 3, pq.Len())
	require.NotNil(t, midItem)

	require.EqualValues(t, "first", pq.Pop().Value)

	pq.Push("before-last", 999)
	require.EqualValues(t, 3, pq.Len())

	pq.Push("after-mid", 501)
	require.EqualValues(t, 4, pq.Len())

	pq.Push("first", 0)
	require.EqualValues(t, 5, pq.Len())

	pq.Push("before-first", 100)
	require.EqualValues(t, 6, pq.Len())

	removed := pq.Remove(midItem)
	require.True(t, removed)
	require.EqualValues(t, 5, pq.Len())

	// Remove again should fail
	removed = pq.Remove(midItem)
	require.False(t, removed)

	pq.Push("first-same-priority", 0)
	require.EqualValues(t, 6, pq.Len())

	require.EqualValues(t, "first", pq.Pop().Value)
	require.EqualValues(t, "first-same-priority", pq.Pop().Value)
	require.EqualValues(t, "before-first", pq.Pop().Value)
	require.EqualValues(t, "after-mid", pq.Pop().Value)
	require.EqualValues(t, "before-last", pq.Pop().Value)
	require.EqualValues(t, "last", pq.Pop().Value)
	require.EqualValues(t, 0, pq.Len())
}
