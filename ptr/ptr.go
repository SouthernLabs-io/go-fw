package ptr

import (
	"fmt"
	"reflect"
)

// FormatPtr will format the value pointed by p using "%v". It will format it as
// "<nil>" if p is nil.
func FormatPtr[T comparable](p *T) string {
	if p == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", *p)
}

// ToPtr returns a pointer to v.
func ToPtr[T any](v T) *T {
	return &v
}

// ToPtrIfNotNil returns a pointer to v if v is not nil. Otherwise, it returns nil. It is reflection based so prefer to use ToPtrIfNotZero for all comparable types.
func ToPtrIfNotNil[T any](v T) *T {
	t := reflect.TypeOf(v)
	if t == nil {
		return nil
	}
	kind := t.Kind()
	if kind == reflect.Slice || kind == reflect.Map || kind == reflect.Chan || kind == reflect.Func {
		val := reflect.ValueOf(v)
		if val.IsNil() {
			return nil
		}
	}
	return &v
}

// ToPtrIfNotZero returns a pointer to v if v is not the zero value for its type.
func ToPtrIfNotZero[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

// ToValue returns the value referenced by p. Returns a zero value if p is nil.
func ToValue[T any](p *T) (v T) {
	if p == nil {
		return v
	}
	return *p
}

// Coalesce returns the first non-nil pointer from the provided list. If all pointers are nil, it returns nil.
func Coalesce[T comparable](ptrs ...*T) *T {
	for _, p := range ptrs {
		if p != nil {
			return p
		}
	}
	return nil
}

func Equal[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
