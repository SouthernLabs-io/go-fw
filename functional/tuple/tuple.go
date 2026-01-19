package tuple

import (
	"fmt"
	"iter"
	"reflect"
	"slices"
	"strings"
)

type Tuple[T1 any, T2 any] struct {
	first  T1
	second T2
}

func New[T1 any, T2 any](first T1, second T2) Tuple[T1, T2] {
	return Tuple[T1, T2]{first: first, second: second}
}

func (t Tuple[T1, T2]) First() T1 {
	return t.first
}

func (t Tuple[T1, T2]) Second() T2 {
	return t.second
}

func (t Tuple[T1, T2]) All() iter.Seq2[int, any] {
	return func(yield func(int, any) bool) {
		_ = yield(0, t.first) && yield(1, t.second)
	}
}

func (t Tuple[T1, T2]) Values() iter.Seq[any] {
	return func(yield func(any) bool) {
		_ = yield(t.first) && yield(t.second)
	}
}

func (t Tuple[T1, T2]) Dim() int {
	return 2
}

func (t Tuple[T1, T2]) At(index int) any {
	switch index {
	case 0:
		return t.first
	case 1:
		return t.second
	default:
		panic("index out of range")
	}
}

func (t Tuple[T1, T2]) String() string {
	return fmt.Sprintf("(%v, %v)", t.first, t.second)
}

func (t Tuple[T1, T2]) GoString() string {
	return fmt.Sprintf("tuple.Tuple[%T, %T]{first: %#v, second: %#v}", t.first, t.second, t.first, t.second)
}

func (t Tuple[T1, T2]) Equals(other Tuple[T1, T2]) bool {
	return reflect.DeepEqual(&t, &other)
}

func (t Tuple[T1, T2]) ToSlice() []any {
	return []any{t.first, t.second}
}

func (t Tuple[T1, T2]) Swap() Tuple[T2, T1] {
	return New(t.second, t.first)
}

func (t Tuple[T1, T2]) Clone() Tuple[T1, T2] {
	return New(t.first, t.second)
}

type Tuple3[T1 any, T2 any, T3 any] struct {
	first  T1
	second T2
	third  T3
}

func New3[T1 any, T2 any, T3 any](first T1, second T2, third T3) Tuple3[T1, T2, T3] {
	return Tuple3[T1, T2, T3]{first: first, second: second, third: third}
}

func (t Tuple3[T1, T2, T3]) First() T1 {
	return t.first
}

func (t Tuple3[T1, T2, T3]) Second() T2 {
	return t.second
}

func (t Tuple3[T1, T2, T3]) Third() T3 {
	return t.third
}

func (t Tuple3[T1, T2, T3]) All() iter.Seq2[int, any] {
	return func(yield func(int, any) bool) {
		_ = yield(0, t.first) && yield(1, t.second) && yield(2, t.third)
	}
}

func (t Tuple3[T1, T2, T3]) Values() iter.Seq[any] {
	return func(yield func(any) bool) {
		_ = yield(t.first) && yield(t.second) && yield(t.third)
	}
}

func (t Tuple3[T1, T2, T3]) Dim() int {
	return 3
}

func (t Tuple3[T1, T2, T3]) At(index int) any {
	switch index {
	case 0:
		return t.first
	case 1:
		return t.second
	case 2:
		return t.third
	default:
		panic(fmt.Sprintf("index out of range: %d, valid: [0,2]", index))
	}
}

func (t Tuple3[T1, T2, T3]) String() string {
	return fmt.Sprintf("(%v, %v, %v)", t.first, t.second, t.third)
}

func (t Tuple3[T1, T2, T3]) GoString() string {
	return fmt.Sprintf("tuple.Tuple3[%T, %T, %T]{first: %#v, second: %#v, third: %#v}", t.first, t.second, t.third, t.first, t.second, t.third)
}

func (t Tuple3[T1, T2, T3]) Equals(other Tuple3[T1, T2, T3]) bool {
	return reflect.DeepEqual(&t, &other)
}

func (t Tuple3[T1, T2, T3]) ToSlice() []any {
	return []any{t.first, t.second, t.third}
}

func (t Tuple3[T1, T2, T3]) Swap() Tuple3[T3, T2, T1] {
	return New3(t.third, t.second, t.first)
}

func (t Tuple3[T1, T2, T3]) Clone() Tuple3[T1, T2, T3] {
	return New3(t.first, t.second, t.third)
}

type TupleN[T any] struct {
	elements []T
}

func NewN[T any](elements ...T) TupleN[T] {
	return TupleN[T]{elements: elements}
}

func (t TupleN[T]) First() T {
	if len(t.elements) == 0 {
		panic("TupleN has no elements")
	}
	return t.elements[0]
}

func (t TupleN[T]) Second() T {
	if len(t.elements) < 2 {
		panic("TupleN has less than 2 elements")
	}
	return t.elements[1]
}

func (t TupleN[T]) Third() T {
	if len(t.elements) < 3 {
		panic("TupleN has less than 3 elements")
	}
	return t.elements[2]
}

func (t TupleN[T]) Fourth() T {
	if len(t.elements) < 4 {
		panic("TupleN has less than 4 elements")
	}
	return t.elements[3]
}

func (t TupleN[T]) Last() T {
	if len(t.elements) == 0 {
		panic("TupleN has no elements")
	}
	return t.elements[len(t.elements)-1]
}

func (t TupleN[T]) At(index int) T {
	if index < 0 || index >= len(t.elements) {
		panic(fmt.Sprintf("index out of range: %d, valid: [0,%d]", index, len(t.elements)-1))
	}
	return t.elements[index]
}

func (t TupleN[T]) Dim() int {
	return len(t.elements)
}

func (t TupleN[T]) All() iter.Seq2[int, T] {
	return slices.All(t.elements)
}

func (t TupleN[T]) Values() iter.Seq[T] {
	return slices.Values(t.elements)
}

func (t TupleN[T]) String() string {
	strs := make([]string, len(t.elements))
	for i, e := range t.elements {
		strs[i] = fmt.Sprintf("%v", e)
	}
	return fmt.Sprintf("(%s)", strings.Join(strs, ", "))
}
func (t TupleN[T]) GoString() string {
	strs := make([]string, len(t.elements))
	for i, e := range t.elements {
		strs[i] = fmt.Sprintf("%#v", e)
	}
	return fmt.Sprintf("tuple.TupleN[%T]{elements: []%T{%s}}", t.elements[0], t.elements[0], strings.Join(strs, ", "))
}

func (t TupleN[T]) Equals(other TupleN[T]) bool {
	return reflect.DeepEqual(&t, &other)
}

func (t TupleN[T]) ToSlice() []T {
	// Clone to prevent external modification
	return slices.Clone(t.elements)
}

func (t TupleN[T]) Swap() TupleN[T] {
	reversed := make([]T, len(t.elements))
	for i, e := range t.elements {
		reversed[len(t.elements)-1-i] = e
	}
	return TupleN[T]{elements: reversed}
}

func (t TupleN[T]) Clone() TupleN[T] {
	return TupleN[T]{elements: slices.Clone(t.elements)}
}
