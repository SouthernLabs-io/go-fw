package syncmap

import "sync"

type Map[K comparable, V any] struct {
	syncMap *sync.Map
	mutex   *sync.Mutex
}

func New[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		syncMap: &sync.Map{},
		mutex:   &sync.Mutex{},
	}
}

func (m *Map[K, V]) Store(key K, value V) {
	m.syncMap.Store(key, value)
}

func (m *Map[K, V]) LoadOrStore(key K, generator func(key K) (value V)) (value V) {
	var present bool
	if value, present = m.Load(key); present {
		return value
	}

	// Lock to avoid calling the generator multiple times for the same key
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if value, present = m.Load(key); present {
		return value
	}
	value = generator(key)
	m.Store(key, value)

	return value
}

func (m *Map[K, V]) Load(key K) (value V, present bool) {
	var v any
	if v, present = m.syncMap.Load(key); present {
		return v.(V), true
	}
	return value, false
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
