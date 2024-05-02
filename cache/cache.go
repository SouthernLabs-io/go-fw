package cache

import (
	"github.com/southernlabs-io/go-fw/errors"
)

var ErrCacheEntryNotFound = errors.Newf("CACHE_ENTRY_NOT_FOUND", "cache entry not found")

type Cache[T any] interface {
	Get(key string, dest *T) error
	Set(key string, value T) error
	Remove(key string) error
}
