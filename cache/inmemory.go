package cache

import (
	"time"

	"github.com/allegro/bigcache"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/southernlabs-io/go-fw/errors"
)

type _InMemoryCache[T any] struct {
	cache      *bigcache.BigCache
	defaultTtl time.Duration
}

func NewInMemoryCache[T any](name string, ttl time.Duration) (Cache[T], error) {
	if ttl < 1 {
		return nil, errors.Newf(errors.ErrCodeBadArgument, "invalid TTL for cache %s: %d, TTL must be a positive duration", name, ttl)
	}
	cache, err := bigcache.NewBigCache(bigcache.DefaultConfig(ttl))
	if err != nil {
		return nil, errors.NewUnknownf("failed to create cache: %s, error: %w", name, err)
	}
	return &_InMemoryCache[T]{cache, ttl}, nil
}

func (c *_InMemoryCache[T]) Get(key string, dest *T) error {
	bytes, err := c.cache.Get(key)
	if err != nil {
		if errors.Is(err, bigcache.ErrEntryNotFound) {
			return ErrCacheEntryNotFound
		}
		return err
	}
	return msgpack.Unmarshal(bytes, dest)
}

func (c *_InMemoryCache[T]) Set(key string, value T) error {
	bytes, err := msgpack.Marshal(value)
	if err != nil {
		return errors.NewUnknownf("could not serialize value for key: %s, error: %w", key, err)
	}
	return c.cache.Set(key, bytes)
}

func (c *_InMemoryCache[T]) Remove(key string) error {
	return c.cache.Delete(key)
}
