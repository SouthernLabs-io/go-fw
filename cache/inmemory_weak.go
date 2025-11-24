package cache

import (
	"fmt"
	"time"

	"github.com/allegro/bigcache"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/southernlabs-io/go-fw/errors"
)

type _InMemoryWeakCache[K comparable, T any] struct {
	cache      *bigcache.BigCache
	defaultTtl time.Duration
}

// NewInMemoryWeakCache creates a new in-memory weak cache with the given name and TTL. It serializes values using msgpack to remove the strong references, but this incurs serialization overhead.
func NewInMemoryWeakCache[K comparable, T any](name string, ttl time.Duration) (Cache[K, T], error) {
	if ttl < 1 {
		return nil, errors.NewBadArgumentf("invalid TTL for cache %s: %d, TTL must be a positive duration", name, ttl)
	}
	cache, err := bigcache.NewBigCache(bigcache.DefaultConfig(ttl))
	if err != nil {
		return nil, errors.NewUnknownf("failed to create cache: %s, error: %w", name, err)
	}
	return &_InMemoryWeakCache[K, T]{cache, ttl}, nil
}

func (c *_InMemoryWeakCache[K, T]) Get(key K) (T, bool, error) {
	var zero T
	kstr := fmt.Sprintf("%v", key)
	bytes, err := c.cache.Get(kstr)
	if err != nil {
		if errors.Is(err, bigcache.ErrEntryNotFound) {
			return zero, false, nil
		}
		return zero, false, errors.NewUnknownf("could not get key from cache: %s, error: %w", kstr, err)
	}
	var v T
	err = msgpack.Unmarshal(bytes, &v)
	if err != nil {
		return zero, false, errors.NewUnknownf("could not deserialize value for key: %s, error: %w", kstr, err)
	}
	return v, true, nil
}

func (c *_InMemoryWeakCache[K, T]) Set(key K, value T) error {
	kstr := fmt.Sprintf("%v", key)
	bytes, err := msgpack.Marshal(value)
	if err != nil {
		return errors.NewUnknownf("could not serialize value for key: %s, error: %w", kstr, err)
	}
	err = c.cache.Set(kstr, bytes)
	if err != nil {
		return errors.NewUnknownf("could not set key in cache: %s, error: %w", kstr, err)
	}
	return nil
}

func (c *_InMemoryWeakCache[K, T]) Del(key K) error {
	kstr := fmt.Sprintf("%v", key)
	err := c.cache.Delete(kstr)
	if err != nil {
		if errors.Is(err, bigcache.ErrEntryNotFound) {
			return nil
		}
		return errors.NewUnknownf("could not delete key from cache: %s, error: %w", kstr, err)
	}
	return nil
}

func (c *_InMemoryWeakCache[K, T]) Clear() error {
	err := c.cache.Reset()
	if err != nil {
		return errors.NewUnknownf("could not clear cache, error: %w", err)
	}
	return nil
}

func (c *_InMemoryWeakCache[K, T]) Len() (int64, error) {
	return int64(c.cache.Len()), nil
}
