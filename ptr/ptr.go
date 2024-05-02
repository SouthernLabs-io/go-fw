package ptr

import "fmt"

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

// ToValue returns the value referenced by p. Returns a zero value if p is nil.
func ToValue[T comparable](p *T) (v T) {
	if p == nil {
		return v
	}
	return *p
}
