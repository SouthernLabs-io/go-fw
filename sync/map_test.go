package sync_test

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/sync"
)

func TestConcurrent(t *testing.T) {
	for range 1_000 {
		testConcurrent(t)
	}
}

func testConcurrent(t *testing.T) {
	m := sync.NewMap[string, string]()
	require.NotNil(t, m)

	genCount := &atomic.Int32{}

	n := 100
	wg := &sync.WaitGroup{}
	wg.Add(n)
	for i := 1; i <= n; i++ {
		var i = i
		go func() {
			key := "key"
			val := m.LoadOrStoreFunc(key, func(key string) string {
				currentCount := genCount.Add(1)
				require.LessOrEqual(t, currentCount, int32(1), "generator called more than once: %d", currentCount)
				return fmt.Sprintf("%d", i)
			})
			require.NotEmpty(t, val)
			val2, present := m.Load(key)
			require.True(t, present)
			require.Equal(t, val, val2)
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestAPI(t *testing.T) {
	m := sync.NewMap[string, string]()
	require.NotNil(t, m)

	key := "key"
	value, present := m.Load(key)
	require.False(t, present)
	require.Equal(t, "", value)

	value = "value"
	m.Store(key, value)
	value2, present := m.Load(key)
	require.True(t, present)
	require.Equal(t, value, value2)

	m.Delete(key)
	value, present = m.Load(key)
	require.False(t, present)
	require.Equal(t, "", value)

	m.Delete(key)
	value, present = m.Load(key)
	require.False(t, present)
	require.Equal(t, "", value)

	value = m.LoadOrStoreFunc(key, func(key string) (value string) {
		return "value"
	})
	require.Equal(t, "value", value)

	m.Clear()
	value, present = m.Load(key)
	require.False(t, present)
	require.Equal(t, "", value)

	m.Store(key, "value")
	m.Store("key2", "value2")
	m.Store("key3", "value3")
	values := m.Values()
	require.Equal(t, 3, len(values))
	require.Contains(t, values, "value")
	require.Contains(t, values, "value2")
	require.Contains(t, values, "value3")

	keys := m.Keys()
	require.Equal(t, 3, len(keys))
	require.Contains(t, keys, "key")
	require.Contains(t, keys, "key2")
	require.Contains(t, keys, "key3")
}

func TestLoadOrStoreVariants(t *testing.T) {
	m := sync.NewMap[string, string]()

	value, loaded := m.LoadOrStore("key", "value")
	require.False(t, loaded)
	require.Equal(t, "value", value)

	value, loaded = m.LoadOrStore("key", "other")
	require.True(t, loaded)
	require.Equal(t, "value", value)
}

func TestLoadOrStoreFuncErr(t *testing.T) {
	m := sync.NewMap[string, string]()
	expectedErr := errors.New("boom")

	value, err := m.LoadOrStoreFuncErr("key", func(key string) (string, error) {
		return "value", expectedErr
	})
	require.ErrorIs(t, err, expectedErr)
	require.Empty(t, value)

	value, present := m.Load("key")
	require.False(t, present)
	require.Empty(t, value)
}
