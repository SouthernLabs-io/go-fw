package slices

import "github.com/southernlabs-io/go-fw/functional/predicates"

func Filter[E any](s []E, predicate predicates.PredicateFunc[E]) []E {
	return FilterI(s, func(_ int, e E) bool {
		return predicate(e)
	})
}

func FilterI[E any](s []E, predicate predicates.PredicateIFunc[E]) []E {
	var filtered []E
	for i, e := range s {
		if predicate(i, e) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func Map[E any, R any](s []E, mapFunc func(E) R) []R {
	return MapI(s, func(_ int, e E) R {
		return mapFunc(e)
	})
}

func MapI[E any, R any](s []E, mapFunc func(int, E) R) []R {
	mapped := make([]R, len(s))
	for i, e := range s {
		mapped[i] = mapFunc(i, e)
	}
	return mapped
}

func MapIE[E any, R any](s []E, mapFunc func(int, E) (R, error)) ([]R, error) {
	mapped := make([]R, len(s))
	var err error
	for i, e := range s {
		mapped[i], err = mapFunc(i, e)
		if err != nil {
			return nil, err
		}
	}
	return mapped, nil
}

func FlatMap[E any, R any](s []E, flatMapFunc func(E) []R) []R {
	return FlatMapI(s, func(_ int, e E) []R {
		return flatMapFunc(e)
	})
}

func FlatMapI[E any, R any](s []E, flatMapFunc func(int, E) []R) []R {
	var mapped []R
	for i, e := range s {
		mapped = append(mapped, flatMapFunc(i, e)...)
	}
	return mapped
}

func FlatMapIE[E any, R any](s []E, flatMapFunc func(int, E) ([]R, error)) ([]R, error) {
	var mapped []R
	for i, e := range s {
		rs, err := flatMapFunc(i, e)
		if err != nil {
			return nil, err
		}
		mapped = append(mapped, rs...)
	}
	return mapped, nil
}

func FindMap[E any, R any](s []E, mapFunc func(e E) (R, bool)) (R, int) {
	return FindMapI(s, func(_ int, e E) (R, bool) {
		return mapFunc(e)
	})
}

func FindMapI[E any, R any](s []E, mapIFunc func(i int, e E) (R, bool)) (R, int) {
	for i, e := range s {
		v, found := mapIFunc(i, e)
		if found {
			return v, i
		}
	}
	var zero R
	return zero, -1
}

func FindMapIE[E any, R any](s []E, mapIFunc func(i int, e E) (R, bool, error)) (R, int, error) {
	var zero R
	for i, e := range s {
		v, found, err := mapIFunc(i, e)
		if err != nil {
			return zero, -1, err
		}
		if found {
			return v, i, nil
		}
	}

	return zero, -1, nil
}

func FindLastMap[E any, R any](s []E, mapFunc func(e E) (R, bool)) (R, int) {
	return FindLastMapI(s, func(_ int, e E) (R, bool) {
		return mapFunc(e)
	})
}
func FindLastMapI[E any, R any](s []E, mapIFunc func(i int, e E) (R, bool)) (R, int) {
	count := len(s)
	for i := count - 1; i > 0; i-- {
		if v, found := mapIFunc(i, s[i]); found {
			return v, i
		}
	}
	var zero R
	return zero, -1
}

func FindLastMapIE[E any, R any](s []E, mapIFunc func(i int, e E) (R, bool, error)) (R, int, error) {
	var zero R
	count := len(s)
	for i := count - 1; i > 0; i-- {
		v, found, err := mapIFunc(i, s[i])
		if err != nil {
			return zero, -1, err
		}
		if found {
			return v, i, nil
		}
	}

	return zero, -1, nil
}
