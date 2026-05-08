package sync

import "sync"

// Map is a thread-safe generic map implementation on top of sync.Map
type Map[K comparable, V any] struct {
	syncMap *sync.Map
	mutex   *sync.Mutex
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		syncMap: &sync.Map{},
		mutex:   &sync.Mutex{},
	}
}

func (m *Map[K, V]) Store(key K, value V) {
	m.syncMap.Store(key, value)
}

// LoadOrStoreFunc loads the value for the given key, if it exists. If the key does not exist,
// it calls the generator function to create the value and stores it in the map.
// The generator function is guaranteed to be called synchronously and only once for the same key.
func (m *Map[K, V]) LoadOrStoreFunc(key K, generator func(key K) (value V)) (value V) {
	val, _ := m.LoadOrStoreFuncErr(key, func(k K) (V, error) {
		return generator(k), nil
	})
	return val
}

// LoadOrStoreFuncErr loads the value for the given key, if it exists. If the key does not exist,
// it calls the generator function to create the value and stores it in the map.
// If the generator function returns an error, the value is not stored and the error is returned.
// The generator function is guaranteed to be called synchronously and only once for the same key.
func (m *Map[K, V]) LoadOrStoreFuncErr(key K, generator func(key K) (value V, err error)) (value V, err error) {
	var present bool
	if value, present = m.Load(key); present {
		return value, nil
	}

	// Lock to avoid calling the generator multiple times for the same key
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if value, present = m.Load(key); present {
		return value, nil
	}
	value, err = generator(key)
	if err != nil {
		var zero V
		return zero, err
	}
	m.Store(key, value)
	return value, nil
}

func (m *Map[K, V]) LoadOrStore(key K, store V) (V, bool) {
	value, loaded := m.syncMap.LoadOrStore(key, store)
	return value.(V), loaded
}

func (m *Map[K, V]) Load(key K) (V, bool) {
	value, present := m.syncMap.Load(key)
	if !present {
		var zero V
		return zero, present
	}
	return value.(V), present
}

func (m *Map[K, V]) Delete(key K) {
	m.syncMap.Delete(key)
}

func (m *Map[K, V]) Clear() {
	m.syncMap.Range(func(key, _ any) bool {
		m.syncMap.Delete(key)
		return true
	})
}

func (m *Map[K, V]) Values() []V {
	var values []V
	m.Range(func(_ K, value V) bool {
		if len(values) == 0 {
			values = []V{}
		}
		values = append(values, value)
		return true
	})
	return values
}

func (m *Map[K, V]) Keys() []K {
	var keys []K
	m.Range(func(key K, _ V) bool {
		if len(keys) == 0 {
			keys = []K{}
		}
		keys = append(keys, key)
		return true
	})
	return keys
}

func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.syncMap.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}
