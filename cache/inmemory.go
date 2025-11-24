package cache

import (
	"time"

	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/sync"
)

type _InMemoryCacheEntry[T any] struct {
	value      T
	expiryTime time.Time
}

type _InMemoryCache[K comparable, T any] struct {
	cache      *sync.Map[K, _InMemoryCacheEntry[T]]
	defaultTtl time.Duration
}

// NewInMemoryCache creates a new in-memory cache with the given name and TTL.
// It is a strong reference cache, meaning that values are kept in memory until explicitly deleted or the cache is cleared.
// It uses a sync.Map for thread-safe operations.
// Expired entries are removed upon access, so to keep the cache size optimal, periodic cleanup is recommended.
func NewInMemoryCache[K comparable, T any](name string, ttl time.Duration) (Cache[K, T], error) {
	if ttl < 1 {
		return nil, errors.NewBadArgumentf("invalid TTL for cache %s: %d, TTL must be a positive duration", name, ttl)
	}
	cache := sync.NewMap[K, _InMemoryCacheEntry[T]]()
	return &_InMemoryCache[K, T]{cache, ttl}, nil
}

func (c *_InMemoryCache[K, T]) Get(key K) (T, bool, error) {
	var zero T
	v, found := c.cache.Load(key)
	if !found {
		return zero, false, nil
	}
	if time.Now().After(v.expiryTime) {
		c.cache.Delete(key)
		return zero, false, nil
	}
	return v.value, true, nil
}

func (c *_InMemoryCache[K, T]) Set(key K, value T) error {
	expiry := time.Now().Add(c.defaultTtl)
	entry := _InMemoryCacheEntry[T]{value: value, expiryTime: expiry}
	c.cache.Store(key, entry)
	return nil
}

func (c *_InMemoryCache[K, T]) Del(key K) error {
	c.cache.Delete(key)
	return nil
}

func (c *_InMemoryCache[K, T]) Clear() error {
	c.cache.Clear()
	return nil
}

func (c *_InMemoryCache[K, T]) Len() (int64, error) {
	var count int64 = 0
	now := time.Now()
	c.cache.Range(func(key K, value _InMemoryCacheEntry[T]) bool {
		if now.After(value.expiryTime) {
			c.cache.Delete(key)
		} else {
			count++
		}
		return true
	})
	return count, nil
}
