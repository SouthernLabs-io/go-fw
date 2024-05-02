package predicates

import (
	"reflect"
	"slices"
)

type PredicateFunc[E any] func(E) bool
type PredicateIFunc[E any] func(int, E) bool

func Not[E any](f PredicateFunc[E]) PredicateFunc[E] {
	return func(e E) bool {
		return !f(e)
	}
}

func NotI[E any](f PredicateIFunc[E]) PredicateIFunc[E] {
	return func(i int, e E) bool {
		return !f(i, e)
	}
}

var _ PredicateFunc[any] = Nil[any]

// Nil is shortcut for NilI(0, e)
func Nil[E any](e E) bool {
	return NilI(0, e)
}

var _ PredicateIFunc[any] = NilI[any]

// NilI is a copy of the internal implementation from stretchr/testify/assert package
func NilI[E any](_ int, e E) bool {
	if any(e) == nil {
		return true
	}

	value := reflect.ValueOf(e)
	kind := value.Kind()
	isNillableKind := slices.Contains(
		[]reflect.Kind{
			reflect.Chan, reflect.Func,
			reflect.Interface, reflect.Map,
			reflect.Ptr, reflect.Slice, reflect.UnsafePointer},
		kind)

	if isNillableKind && value.IsNil() {
		return true
	}

	return false
}

var _ PredicateFunc[any] = Empty[any]

// Empty is shortcut for EmptyI(0, e)
func Empty[E any](e E) bool {
	return EmptyI(0, e)
}

var _ PredicateIFunc[any] = EmptyI[any]

// EmptyI is a copy of the internal implementation from stretchr/testify/assert package
func EmptyI[E any](_ int, e E) bool {
	// get nil case out of the way
	if any(e) == nil {
		return true
	}

	objValue := reflect.ValueOf(e)

	switch objValue.Kind() {
	// collection types are empty when they have no element
	case reflect.Chan, reflect.Map, reflect.Slice:
		return objValue.Len() == 0
	// pointers are empty if nil or if the value they point to is empty
	case reflect.Ptr:
		if objValue.IsNil() {
			return true
		}
		deref := objValue.Elem().Interface()
		return EmptyI(0, deref)
	// for all other types, compare against the zero value
	// array types are empty when they match their zero-initialized state
	default:
		zero := reflect.Zero(objValue.Type())
		return reflect.DeepEqual(e, zero.Interface())
	}
}
