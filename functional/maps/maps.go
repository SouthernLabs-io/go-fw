package maps

import "github.com/southernlabs-io/go-fw/functional/predicates"

type Entry[K comparable, V any] struct {
	Key   K
	Value V
}

func ToSlice[K comparable, V any](m map[K]V) []Entry[K, V] {
	entries := make([]Entry[K, V], len(m))
	i := 0
	for k, v := range m {
		entries[i] = Entry[K, V]{k, v}
		i++
	}
	return entries
}

func Filter[K comparable, V any](m map[K]V, predicate predicates.PredicateFunc[Entry[K, V]]) map[K]V {
	filtered := make(map[K]V)
	for k, v := range m {
		if predicate(Entry[K, V]{k, v}) {
			filtered[k] = v
		}
	}
	return filtered
}

func Map[K comparable, K2 comparable, V any, V2 any](m map[K]V, mapFunc func(Entry[K, V]) Entry[K2, V2]) map[K2]V2 {
	mapped, _ := MapE(m, func(entry Entry[K, V]) (Entry[K2, V2], error) {
		return mapFunc(entry), nil
	})
	return mapped
}

func MapE[K comparable, K2 comparable, V any, V2 any](m map[K]V, mapFunc func(Entry[K, V]) (Entry[K2, V2], error)) (map[K2]V2, error) {
	mapped := make(map[K2]V2)
	for k, v := range m {
		newEntry, err := mapFunc(Entry[K, V]{k, v})
		if err != nil {
			return nil, err
		}
		mapped[newEntry.Key] = newEntry.Value
	}
	return mapped, nil
}

func FromEntries[K comparable, V any](entries []Entry[K, V]) map[K]V {
	m := make(map[K]V)
	for _, entry := range entries {
		m[entry.Key] = entry.Value
	}
	return m
}
