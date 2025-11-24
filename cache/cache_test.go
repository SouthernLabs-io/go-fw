package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/errors"
)

type testValue struct {
	Name  string
	Value int
}

func TestCacheInterface(t *testing.T) {
	testCases := []struct {
		name    string
		factory func() (Cache[string, int], error)
	}{
		{
			name: "InMemoryCache",
			factory: func() (Cache[string, int], error) {
				return NewInMemoryCache[string, int]("test", time.Hour)
			},
		},
		{
			name: "InMemoryWeakCache",
			factory: func() (Cache[string, int], error) {
				return NewInMemoryWeakCache[string, int]("test", time.Hour)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache, err := tc.factory()
			require.NoError(t, err)
			require.NotNil(t, cache)

			// Test initial state
			len, err := cache.Len()
			require.NoError(t, err)
			require.Equal(t, int64(0), len)

			// Test Get non-existent key
			val, found, err := cache.Get("nonexistent")
			require.NoError(t, err)
			require.False(t, found)
			require.Equal(t, 0, val)

			// Test Set and Get
			err = cache.Set("key1", 42)
			require.NoError(t, err)

			val, found, err = cache.Get("key1")
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, 42, val)

			// Test Len after set
			len, err = cache.Len()
			require.NoError(t, err)
			require.Equal(t, int64(1), len)

			// Test Set another key
			err = cache.Set("key2", 100)
			require.NoError(t, err)

			len, err = cache.Len()
			require.NoError(t, err)
			require.Equal(t, int64(2), len)

			val, found, err = cache.Get("key2")
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, 100, val)

			// Test overwrite
			err = cache.Set("key1", 43)
			require.NoError(t, err)

			val, found, err = cache.Get("key1")
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, 43, val)

			len, err = cache.Len()
			require.NoError(t, err)
			require.Equal(t, int64(2), len)

			// Test Del non-existent key
			err = cache.Del("nonexistent")
			require.NoError(t, err)

			len, err = cache.Len()
			require.NoError(t, err)
			require.Equal(t, int64(2), len)

			// Test Del existing key
			err = cache.Del("key1")
			require.NoError(t, err)

			val, found, err = cache.Get("key1")
			require.NoError(t, err)
			require.False(t, found)
			require.Equal(t, 0, val)

			len, err = cache.Len()
			require.NoError(t, err)
			require.Equal(t, int64(1), len)

			// Test Clear
			err = cache.Clear()
			require.NoError(t, err)

			len, err = cache.Len()
			require.NoError(t, err)
			require.Equal(t, int64(0), len)

			val, found, err = cache.Get("key2")
			require.NoError(t, err)
			require.False(t, found)
			require.Equal(t, 0, val)
		})
	}
}

func TestCacheExpiry(t *testing.T) {
	// Only test inmemory cache for expiry, as weak cache uses bigcache which handles expiry internally
	cache, err := NewInMemoryCache[string, int]("test", 100*time.Millisecond)
	require.NoError(t, err)

	err = cache.Set("key", 42)
	require.NoError(t, err)

	val, found, err := cache.Get("key")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 42, val)

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	val, found, err = cache.Get("key")
	require.NoError(t, err)
	require.False(t, found)
	require.Equal(t, 0, val)

	// Len should clean up expired entries
	len, err := cache.Len()
	require.NoError(t, err)
	require.Equal(t, int64(0), len)
}

func TestCacheComplexTypes(t *testing.T) {
	testCases := []struct {
		name      string
		factory   func() (Cache[string, testValue], error)
		cacheType string
	}{
		{
			name: "InMemoryCache",
			factory: func() (Cache[string, testValue], error) {
				return NewInMemoryCache[string, testValue]("test", time.Hour)
			},
			cacheType: "inmemory",
		},
		{
			name: "InMemoryWeakCache",
			factory: func() (Cache[string, testValue], error) {
				return NewInMemoryWeakCache[string, testValue]("test", time.Hour)
			},
			cacheType: "weak",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache, err := tc.factory()
			require.NoError(t, err)

			value := testValue{Name: "test", Value: 123}
			err = cache.Set("key", value)
			require.NoError(t, err)

			retrieved, found, err := cache.Get("key")
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, value, retrieved)
		})
	}
}

func TestCacheIntKeys(t *testing.T) {
	testCases := []struct {
		name      string
		factory   func() (Cache[int, string], error)
		cacheType string
	}{
		{
			name: "InMemoryCache",
			factory: func() (Cache[int, string], error) {
				return NewInMemoryCache[int, string]("test", time.Hour)
			},
			cacheType: "inmemory",
		},
		{
			name: "InMemoryWeakCache",
			factory: func() (Cache[int, string], error) {
				return NewInMemoryWeakCache[int, string]("test", time.Hour)
			},
			cacheType: "weak",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache, err := tc.factory()
			require.NoError(t, err)

			err = cache.Set(42, "value")
			require.NoError(t, err)

			val, found, err := cache.Get(42)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, "value", val)
		})
	}
}

func TestCacheInvalidTTL(t *testing.T) {
	_, err := NewInMemoryCache[string, int]("test", 0)
	require.Error(t, err)
	require.True(t, errors.IsCode(err, errors.ErrCodeBadArgument))

	_, err = NewInMemoryWeakCache[string, int]("test", 0)
	require.Error(t, err)
	require.True(t, errors.IsCode(err, errors.ErrCodeBadArgument))
}

func TestCacheConcurrentAccess(t *testing.T) {
	cache, err := NewInMemoryCache[string, int]("test", time.Hour)
	require.NoError(t, err)

	// Simple concurrent test
	done := make(chan bool, 2)

	go func() {
		for i := range 100 {
			cache.Set("key", i)
		}
		done <- true
	}()

	go func() {
		for range 100 {
			cache.Get("key")
		}
		done <- true
	}()

	<-done
	<-done

	// Should not panic and basic functionality should work
	val, found, err := cache.Get("key")
	require.NoError(t, err)
	require.True(t, found)
	require.True(t, val >= 0 && val < 100)
}
