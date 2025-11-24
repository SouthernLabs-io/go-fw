package cache

type Cache[K comparable, T any] interface {
	Get(key K) (T, bool, error)
	Set(key K, value T) error
	Del(key K) error
	Clear() error
	Len() (int64, error)
}
