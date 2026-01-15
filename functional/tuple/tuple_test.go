package tuple_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/functional/tuple"
)

func TestNewTuple(t *testing.T) {
	t1 := tuple.NewTuple(1, "hello")
	require.NotNil(t, t1)
	require.Equal(t, 1, t1.First())
	require.Equal(t, "hello", t1.Second())
}

func TestTupleFirst(t *testing.T) {
	t1 := tuple.NewTuple(42, "world")
	require.Equal(t, 42, t1.First())

	t2 := tuple.NewTuple("foo", 3.14)
	require.Equal(t, "foo", t2.First())
}

func TestTupleSecond(t *testing.T) {
	t1 := tuple.NewTuple(42, "world")
	require.Equal(t, "world", t1.Second())

	t2 := tuple.NewTuple("foo", 3.14)
	require.Equal(t, 3.14, t2.Second())
}

func TestTupleDim(t *testing.T) {
	t1 := tuple.NewTuple(1, 2)
	require.Equal(t, 2, t1.Dim())

	t2 := tuple.NewTuple("a", "b")
	require.Equal(t, 2, t2.Dim())
}

func TestTupleAt(t *testing.T) {
	t1 := tuple.NewTuple(100, "test")

	val0 := t1.At(0)
	require.Equal(t, 100, val0)

	val1 := t1.At(1)
	require.Equal(t, "test", val1)
}

func TestTupleAtPanic(t *testing.T) {
	t1 := tuple.NewTuple(1, 2)

	require.Panics(t, func() {
		t1.At(2)
	})

	require.Panics(t, func() {
		t1.At(-1)
	})

	require.Panics(t, func() {
		t1.At(100)
	})
}

func TestTupleAll(t *testing.T) {
	t1 := tuple.NewTuple("first", "second")

	count := 0
	for idx, val := range t1.All() {
		count++
		if idx == 0 {
			require.Equal(t, "first", val)
		} else if idx == 1 {
			require.Equal(t, "second", val)
		}
	}
	require.Equal(t, 2, count)
}

func TestTupleValues(t *testing.T) {
	t1 := tuple.NewTuple(10, 20)

	values := make([]any, 0)
	for val := range t1.Values() {
		values = append(values, val)
	}

	require.Equal(t, 2, len(values))
	require.Equal(t, 10, values[0])
	require.Equal(t, 20, values[1])
}

func TestNewTuple3(t *testing.T) {
	t3 := tuple.NewTuple3(1, "hello", 3.14)
	require.NotNil(t, t3)
	require.Equal(t, 1, t3.First())
	require.Equal(t, "hello", t3.Second())
	require.Equal(t, 3.14, t3.Third())
}

func TestTuple3First(t *testing.T) {
	t3 := tuple.NewTuple3(42, "world", true)
	require.Equal(t, 42, t3.First())

	t4 := tuple.NewTuple3("foo", 3.14, 100)
	require.Equal(t, "foo", t4.First())
}

func TestTuple3Second(t *testing.T) {
	t3 := tuple.NewTuple3(42, "world", true)
	require.Equal(t, "world", t3.Second())

	t4 := tuple.NewTuple3("foo", 3.14, 100)
	require.Equal(t, 3.14, t4.Second())
}

func TestTuple3Third(t *testing.T) {
	t3 := tuple.NewTuple3(42, "world", true)
	require.Equal(t, true, t3.Third())

	t4 := tuple.NewTuple3("foo", 3.14, 100)
	require.Equal(t, 100, t4.Third())
}

func TestTuple3Dim(t *testing.T) {
	t3 := tuple.NewTuple3(1, 2, 3)
	require.Equal(t, 3, t3.Dim())

	t4 := tuple.NewTuple3("a", "b", "c")
	require.Equal(t, 3, t4.Dim())
}

func TestTuple3At(t *testing.T) {
	t3 := tuple.NewTuple3(100, "test", true)

	val0 := t3.At(0)
	require.Equal(t, 100, val0)

	val1 := t3.At(1)
	require.Equal(t, "test", val1)

	val2 := t3.At(2)
	require.Equal(t, true, val2)
}

func TestTuple3AtPanic(t *testing.T) {
	t3 := tuple.NewTuple3(1, 2, 3)

	require.Panics(t, func() {
		t3.At(3)
	})

	require.Panics(t, func() {
		t3.At(-1)
	})

	require.Panics(t, func() {
		t3.At(100)
	})
}

func TestTuple3All(t *testing.T) {
	t3 := tuple.NewTuple3("first", "second", "third")

	count := 0
	for idx, val := range t3.All() {
		count++
		if idx == 0 {
			require.Equal(t, "first", val)
		} else if idx == 1 {
			require.Equal(t, "second", val)
		} else if idx == 2 {
			require.Equal(t, "third", val)
		}
	}
	require.Equal(t, 3, count)
}

func TestTuple3Values(t *testing.T) {
	t3 := tuple.NewTuple3(10, 20, 30)

	values := make([]any, 0)
	for val := range t3.Values() {
		values = append(values, val)
	}

	require.Equal(t, 3, len(values))
	require.Equal(t, 10, values[0])
	require.Equal(t, 20, values[1])
	require.Equal(t, 30, values[2])
}

func TestNewTupleN(t *testing.T) {
	tn := tuple.NewTupleN(1, 2, 3, 4, 5)
	require.NotNil(t, tn)
	require.Equal(t, 5, tn.Dim())
}

func TestTupleNFirst(t *testing.T) {
	tn := tuple.NewTupleN(10, 20, 30, 40)
	require.Equal(t, 10, tn.First())

	tn2 := tuple.NewTupleN("a", "b", "c")
	require.Equal(t, "a", tn2.First())
}

func TestTupleNSecond(t *testing.T) {
	tn := tuple.NewTupleN(10, 20, 30, 40)
	require.Equal(t, 20, tn.Second())

	tn2 := tuple.NewTupleN("a", "b", "c")
	require.Equal(t, "b", tn2.Second())
}

func TestTupleNThird(t *testing.T) {
	tn := tuple.NewTupleN(10, 20, 30, 40)
	require.Equal(t, 30, tn.Third())

	tn2 := tuple.NewTupleN("a", "b", "c", "d")
	require.Equal(t, "c", tn2.Third())
}

func TestTupleNFourth(t *testing.T) {
	tn := tuple.NewTupleN(10, 20, 30, 40, 50)
	require.Equal(t, 40, tn.Fourth())

	tn2 := tuple.NewTupleN("a", "b", "c", "d", "e")
	require.Equal(t, "d", tn2.Fourth())
}

func TestTupleNLast(t *testing.T) {
	tn := tuple.NewTupleN(10, 20, 30, 40, 50)
	require.Equal(t, 50, tn.Last())

	tn2 := tuple.NewTupleN("a", "b", "c")
	require.Equal(t, "c", tn2.Last())

	tn3 := tuple.NewTupleN(100)
	require.Equal(t, 100, tn3.Last())
}

func TestTupleNAt(t *testing.T) {
	tn := tuple.NewTupleN(1, 2, 3, 4, 5)

	require.Equal(t, 1, tn.At(0))
	require.Equal(t, 2, tn.At(1))
	require.Equal(t, 3, tn.At(2))
	require.Equal(t, 4, tn.At(3))
	require.Equal(t, 5, tn.At(4))
}

func TestTupleNDim(t *testing.T) {
	tn1 := tuple.NewTupleN(1)
	require.Equal(t, 1, tn1.Dim())

	tn3 := tuple.NewTupleN(1, 2, 3)
	require.Equal(t, 3, tn3.Dim())

	tn10 := tuple.NewTupleN(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	require.Equal(t, 10, tn10.Dim())
}

func TestTupleNAll(t *testing.T) {
	tn := tuple.NewTupleN(10, 20, 30, 40, 50)

	count := 0
	expectedValues := []int{10, 20, 30, 40, 50}
	for idx, val := range tn.All() {
		require.Equal(t, expectedValues[idx], val)
		count++
	}
	require.Equal(t, 5, count)
}

func TestTupleNValues(t *testing.T) {
	tn := tuple.NewTupleN("a", "b", "c", "d")

	values := make([]string, 0)
	for val := range tn.Values() {
		values = append(values, val)
	}

	require.Equal(t, 4, len(values))
	require.Equal(t, []string{"a", "b", "c", "d"}, values)
}

func TestTupleNEmpty(t *testing.T) {
	tn := tuple.NewTupleN[int]()
	require.Equal(t, 0, tn.Dim())
}

func TestTupleWithDifferentTypes(t *testing.T) {
	// Test with struct types
	type Person struct {
		Name string
		Age  int
	}

	person := Person{Name: "John", Age: 30}
	t1 := tuple.NewTuple(person, "metadata")
	require.Equal(t, person, t1.First())
	require.Equal(t, "metadata", t1.Second())

	// Test with pointer types
	t2 := tuple.NewTuple(&person, 42)
	require.Equal(t, &person, t2.First())
	require.Equal(t, 42, t2.Second())

	// Test with nil values
	t3 := tuple.NewTuple[*string, *int](nil, nil)
	require.Nil(t, t3.First())
	require.Nil(t, t3.Second())
}

func TestTuple3WithDifferentTypes(t *testing.T) {
	// Test with mixed types
	t3 := tuple.NewTuple3([]int{1, 2, 3}, map[string]int{"a": 1}, true)
	require.Equal(t, []int{1, 2, 3}, t3.First())
	require.Equal(t, map[string]int{"a": 1}, t3.Second())
	require.Equal(t, true, t3.Third())
}

func TestTupleNWithSingleElement(t *testing.T) {
	tn := tuple.NewTupleN(42)
	require.Equal(t, 1, tn.Dim())
	require.Equal(t, 42, tn.First())
	require.Equal(t, 42, tn.Last())
	require.Equal(t, 42, tn.At(0))
}

func TestTupleIterationBreak(t *testing.T) {
	t1 := tuple.NewTuple(1, 2)

	// Test early break in All()
	count := 0
	for range t1.All() {
		count++
		break
	}
	require.Equal(t, 1, count)

	// Test early break in Values()
	count = 0
	for range t1.Values() {
		count++
		break
	}
	require.Equal(t, 1, count)
}

func TestTuple3IterationBreak(t *testing.T) {
	t3 := tuple.NewTuple3(1, 2, 3)

	// Test early break in All()
	count := 0
	for range t3.All() {
		count++
		if count == 2 {
			break
		}
	}
	require.Equal(t, 2, count)

	// Test early break in Values()
	count = 0
	for range t3.Values() {
		count++
		if count == 2 {
			break
		}
	}
	require.Equal(t, 2, count)
}

func TestTupleNIterationBreak(t *testing.T) {
	tn := tuple.NewTupleN(1, 2, 3, 4, 5)

	// Test early break in All()
	count := 0
	for range tn.All() {
		count++
		if count == 3 {
			break
		}
	}
	require.Equal(t, 3, count)

	// Test early break in Values()
	count = 0
	for range tn.Values() {
		count++
		if count == 3 {
			break
		}
	}
	require.Equal(t, 3, count)
}

// Tests for new methods

func TestTupleString(t *testing.T) {
	t1 := tuple.NewTuple(42, "hello")
	require.Equal(t, "(42, hello)", t1.String())

	t2 := tuple.NewTuple(3.14, true)
	require.Equal(t, "(3.14, true)", t2.String())

	t3 := tuple.NewTuple("foo", "bar")
	require.Equal(t, "(foo, bar)", t3.String())
}

func TestTupleGoString(t *testing.T) {
	t1 := tuple.NewTuple(42, "hello")
	goStr := t1.GoString()
	require.Contains(t, goStr, "tuple.Tuple")
	require.Contains(t, goStr, "first: 42")
	require.Contains(t, goStr, `second: "hello"`)
}

func TestTupleEquals(t *testing.T) {
	t1 := tuple.NewTuple(42, "hello")
	t2 := tuple.NewTuple(42, "hello")
	t3 := tuple.NewTuple(42, "world")
	t4 := tuple.NewTuple(100, "hello")

	require.True(t, t1.Equals(t2))
	require.False(t, t1.Equals(t3))
	require.False(t, t1.Equals(t4))

	// Test same reference (identity)
	require.True(t, t1.Equals(t1))

	// Test with pointers
	val := 10
	p1 := tuple.NewTuple(&val, &val)
	p2 := tuple.NewTuple(&val, &val)
	require.True(t, p1.Equals(p2))
}

func TestTupleToSlice(t *testing.T) {
	t1 := tuple.NewTuple(42, "hello")
	slice := t1.ToSlice()

	require.Equal(t, 2, len(slice))
	require.Equal(t, 42, slice[0])
	require.Equal(t, "hello", slice[1])

	// Test with different types
	t2 := tuple.NewTuple([]int{1, 2, 3}, map[string]int{"a": 1})
	slice2 := t2.ToSlice()
	require.Equal(t, 2, len(slice2))
	require.Equal(t, []int{1, 2, 3}, slice2[0])
	require.Equal(t, map[string]int{"a": 1}, slice2[1])
}

func TestTupleSwap(t *testing.T) {
	t1 := tuple.NewTuple(42, "hello")
	swapped := t1.Swap()

	require.Equal(t, "hello", swapped.First())
	require.Equal(t, 42, swapped.Second())

	// Original should be unchanged
	require.Equal(t, 42, t1.First())
	require.Equal(t, "hello", t1.Second())

	// Test with different types
	t2 := tuple.NewTuple(3.14, true)
	swapped2 := t2.Swap()
	require.Equal(t, true, swapped2.First())
	require.Equal(t, 3.14, swapped2.Second())
}

func TestTupleClone(t *testing.T) {
	t1 := tuple.NewTuple(42, "hello")
	cloned := t1.Clone()

	require.Equal(t, t1.First(), cloned.First())
	require.Equal(t, t1.Second(), cloned.Second())
	require.True(t, t1.Equals(cloned))
}

func TestTuple3String(t *testing.T) {
	t3 := tuple.NewTuple3(42, "hello", true)
	require.Equal(t, "(42, hello, true)", t3.String())

	t4 := tuple.NewTuple3(1, 2, 3)
	require.Equal(t, "(1, 2, 3)", t4.String())
}

func TestTuple3GoString(t *testing.T) {
	t3 := tuple.NewTuple3(42, "hello", true)
	goStr := t3.GoString()
	require.Contains(t, goStr, "tuple.Tuple3")
	require.Contains(t, goStr, "first: 42")
	require.Contains(t, goStr, `second: "hello"`)
	require.Contains(t, goStr, "third: true")
}

func TestTuple3Equals(t *testing.T) {
	t1 := tuple.NewTuple3(42, "hello", true)
	t2 := tuple.NewTuple3(42, "hello", true)
	t3 := tuple.NewTuple3(42, "hello", false)
	t4 := tuple.NewTuple3(100, "hello", true)

	require.True(t, t1.Equals(t2))
	require.False(t, t1.Equals(t3))
	require.False(t, t1.Equals(t4))

	// Test same reference (identity)
	require.True(t, t1.Equals(t1))
}

func TestTuple3ToSlice(t *testing.T) {
	t3 := tuple.NewTuple3(42, "hello", true)
	slice := t3.ToSlice()

	require.Equal(t, 3, len(slice))
	require.Equal(t, 42, slice[0])
	require.Equal(t, "hello", slice[1])
	require.Equal(t, true, slice[2])
}

func TestTuple3Swap(t *testing.T) {
	t3 := tuple.NewTuple3(42, "hello", true)
	swapped := t3.Swap()

	require.Equal(t, true, swapped.First())
	require.Equal(t, "hello", swapped.Second())
	require.Equal(t, 42, swapped.Third())

	// Original should be unchanged
	require.Equal(t, 42, t3.First())
	require.Equal(t, "hello", t3.Second())
	require.Equal(t, true, t3.Third())
}

func TestTuple3Clone(t *testing.T) {
	t3 := tuple.NewTuple3(42, "hello", true)
	cloned := t3.Clone()

	require.Equal(t, t3.First(), cloned.First())
	require.Equal(t, t3.Second(), cloned.Second())
	require.Equal(t, t3.Third(), cloned.Third())
	require.True(t, t3.Equals(cloned))
}

func TestTuple3AtPanicWithMessage(t *testing.T) {
	t3 := tuple.NewTuple3(1, 2, 3)

	require.PanicsWithValue(t, "index out of range: 5, valid: [0,2]", func() {
		t3.At(5)
	})

	require.PanicsWithValue(t, "index out of range: -1, valid: [0,2]", func() {
		t3.At(-1)
	})
}

func TestTupleNString(t *testing.T) {
	tn1 := tuple.NewTupleN(1, 2, 3, 4, 5)
	require.Equal(t, "(1, 2, 3, 4, 5)", tn1.String())

	tn2 := tuple.NewTupleN("a", "b", "c")
	require.Equal(t, "(a, b, c)", tn2.String())

	tn3 := tuple.NewTupleN(42)
	require.Equal(t, "(42)", tn3.String())
}

func TestTupleNGoString(t *testing.T) {
	tn := tuple.NewTupleN(1, 2, 3)
	goStr := tn.GoString()
	require.Contains(t, goStr, "tuple.TupleN")
	require.Contains(t, goStr, "elements:")
}

func TestTupleNEquals(t *testing.T) {
	tn1 := tuple.NewTupleN(1, 2, 3, 4, 5)
	tn2 := tuple.NewTupleN(1, 2, 3, 4, 5)
	tn3 := tuple.NewTupleN(1, 2, 3, 4, 6)
	tn4 := tuple.NewTupleN(1, 2, 3)

	require.True(t, tn1.Equals(tn2))
	require.False(t, tn1.Equals(tn3))
	require.False(t, tn1.Equals(tn4))

	// Test same reference (identity)
	require.True(t, tn1.Equals(tn1))

	// Test with strings
	tns1 := tuple.NewTupleN("a", "b", "c")
	tns2 := tuple.NewTupleN("a", "b", "c")
	require.True(t, tns1.Equals(tns2))
}

func TestTupleNToSlice(t *testing.T) {
	tn := tuple.NewTupleN(1, 2, 3, 4, 5)
	slice := tn.ToSlice()

	require.Equal(t, 5, len(slice))
	require.Equal(t, []int{1, 2, 3, 4, 5}, slice)

	// Verify it's a clone - modifying slice shouldn't affect tuple
	slice[0] = 999
	require.Equal(t, 1, tn.First())
}

func TestTupleNSwap(t *testing.T) {
	tn := tuple.NewTupleN(1, 2, 3, 4, 5)
	swapped := tn.Swap()

	require.Equal(t, 5, swapped.First())
	require.Equal(t, 4, swapped.Second())
	require.Equal(t, 3, swapped.Third())
	require.Equal(t, 2, swapped.Fourth())
	require.Equal(t, 1, swapped.Last())

	// Original should be unchanged
	require.Equal(t, 1, tn.First())
	require.Equal(t, 5, tn.Last())

	// Test with even number of elements
	tn2 := tuple.NewTupleN(10, 20, 30, 40)
	swapped2 := tn2.Swap()
	require.Equal(t, 40, swapped2.First())
	require.Equal(t, 30, swapped2.Second())
	require.Equal(t, 20, swapped2.Third())
	require.Equal(t, 10, swapped2.Fourth())
}

func TestTupleNClone(t *testing.T) {
	tn := tuple.NewTupleN(1, 2, 3, 4, 5)
	cloned := tn.Clone()

	require.Equal(t, tn.Dim(), cloned.Dim())
	require.True(t, tn.Equals(cloned))

	// Verify they are independent - get slices and modify
	originalSlice := tn.ToSlice()
	clonedSlice := cloned.ToSlice()
	originalSlice[0] = 999
	require.NotEqual(t, originalSlice, clonedSlice)
}

func TestTupleNFirstPanic(t *testing.T) {
	tn := tuple.NewTupleN[int]()
	require.PanicsWithValue(t, "TupleN has no elements", func() {
		tn.First()
	})
}

func TestTupleNSecondPanic(t *testing.T) {
	tn := tuple.NewTupleN(1)
	require.PanicsWithValue(t, "TupleN has less than 2 elements", func() {
		tn.Second()
	})

	tn2 := tuple.NewTupleN[int]()
	require.PanicsWithValue(t, "TupleN has less than 2 elements", func() {
		tn2.Second()
	})
}

func TestTupleNThirdPanic(t *testing.T) {
	tn := tuple.NewTupleN(1, 2)
	require.PanicsWithValue(t, "TupleN has less than 3 elements", func() {
		tn.Third()
	})
}

func TestTupleNFourthPanic(t *testing.T) {
	tn := tuple.NewTupleN(1, 2, 3)
	require.PanicsWithValue(t, "TupleN has less than 4 elements", func() {
		tn.Fourth()
	})
}

func TestTupleNLastPanic(t *testing.T) {
	tn := tuple.NewTupleN[int]()
	require.PanicsWithValue(t, "TupleN has no elements", func() {
		tn.Last()
	})
}

func TestTupleNAtPanicWithMessage(t *testing.T) {
	tn := tuple.NewTupleN(1, 2, 3, 4, 5)

	require.PanicsWithValue(t, "index out of range: 10, valid: [0,4]", func() {
		tn.At(10)
	})

	require.PanicsWithValue(t, "index out of range: -1, valid: [0,4]", func() {
		tn.At(-1)
	})

	// Test with empty tuple
	tn2 := tuple.NewTupleN[int]()
	require.PanicsWithValue(t, "index out of range: 0, valid: [0,-1]", func() {
		tn2.At(0)
	})
}

func TestTupleNSwapSingleElement(t *testing.T) {
	tn := tuple.NewTupleN(42)
	swapped := tn.Swap()
	require.Equal(t, 42, swapped.First())
	require.Equal(t, 1, swapped.Dim())
}

func TestTupleNSwapTwoElements(t *testing.T) {
	tn := tuple.NewTupleN("first", "second")
	swapped := tn.Swap()
	require.Equal(t, "second", swapped.First())
	require.Equal(t, "first", swapped.Second())
}

// Immutability tests

func TestTupleImmutabilityWithMutableTypes(t *testing.T) {
	// Test with slice field
	slice1 := []int{1, 2, 3}
	slice2 := []int{4, 5, 6}
	t1 := tuple.NewTuple(slice1, slice2)

	// Modify original slices
	slice1[0] = 999
	slice2[0] = 888

	// Tuple should still reference the same slices (shallow copy)
	// This is expected behavior - Tuple doesn't deep copy
	require.Equal(t, 999, t1.First()[0])
	require.Equal(t, 888, t1.Second()[0])
}

func TestTupleSwapImmutability(t *testing.T) {
	original := tuple.NewTuple(42, "hello")
	swapped := original.Swap()

	// Verify original is unchanged
	require.Equal(t, 42, original.First())
	require.Equal(t, "hello", original.Second())

	// Verify swapped has reversed values
	require.Equal(t, "hello", swapped.First())
	require.Equal(t, 42, swapped.Second())
}

func TestTupleCloneImmutability(t *testing.T) {
	original := tuple.NewTuple(42, "hello")
	cloned := original.Clone()

	// They should be equal but independent
	require.True(t, original.Equals(cloned))

	// With value types, they're already independent
	// Test with mutable types
	slice1 := []int{1, 2, 3}
	slice2 := []int{4, 5, 6}
	t1 := tuple.NewTuple(slice1, slice2)
	t2 := t1.Clone()

	// Both reference same underlying slices (shallow copy)
	slice1[0] = 999
	require.Equal(t, 999, t1.First()[0])
	require.Equal(t, 999, t2.First()[0]) // Also affected
}

func TestTuple3ImmutabilityWithMutableTypes(t *testing.T) {
	// Test with map field
	map1 := map[string]int{"a": 1}
	map2 := map[string]int{"b": 2}
	slice1 := []int{1, 2, 3}
	t3 := tuple.NewTuple3(map1, map2, slice1)

	// Modify original map and slice
	map1["a"] = 999
	slice1[0] = 888

	// Tuple should still reference the same map/slice (shallow copy)
	require.Equal(t, 999, t3.First()["a"])
	require.Equal(t, 888, t3.Third()[0])
}

func TestTuple3SwapImmutability(t *testing.T) {
	original := tuple.NewTuple3(1, 2, 3)
	swapped := original.Swap()

	// Verify original is unchanged
	require.Equal(t, 1, original.First())
	require.Equal(t, 2, original.Second())
	require.Equal(t, 3, original.Third())

	// Verify swapped has reversed values
	require.Equal(t, 3, swapped.First())
	require.Equal(t, 2, swapped.Second())
	require.Equal(t, 1, swapped.Third())
}

func TestTuple3CloneImmutability(t *testing.T) {
	original := tuple.NewTuple3(1, "hello", true)
	cloned := original.Clone()

	// They should be equal but independent for value types
	require.True(t, original.Equals(cloned))
}

func TestTupleNImmutabilityWithMutableSlice(t *testing.T) {
	// Create tuple with slice elements
	tn := tuple.NewTupleN([]int{1, 2, 3}, []int{4, 5, 6}, []int{7, 8, 9})

	// Get first element and modify it
	first := tn.First()
	first[0] = 999

	// This WILL affect the tuple since it's a shallow copy
	require.Equal(t, 999, tn.First()[0])
}

func TestTupleNToSliceImmutability(t *testing.T) {
	tn := tuple.NewTupleN(1, 2, 3, 4, 5)
	slice := tn.ToSlice()

	// Modify the returned slice
	slice[0] = 999
	slice[4] = 888

	// Original tuple should be unchanged (ToSlice returns a clone)
	require.Equal(t, 1, tn.First())
	require.Equal(t, 5, tn.Last())
}

func TestTupleNSwapImmutability(t *testing.T) {
	original := tuple.NewTupleN(1, 2, 3, 4, 5)
	swapped := original.Swap()

	// Verify original is unchanged
	require.Equal(t, 1, original.First())
	require.Equal(t, 2, original.Second())
	require.Equal(t, 5, original.Last())

	// Verify swapped has reversed values
	require.Equal(t, 5, swapped.First())
	require.Equal(t, 4, swapped.Second())
	require.Equal(t, 1, swapped.Last())
}

func TestTupleNCloneImmutability(t *testing.T) {
	original := tuple.NewTupleN(1, 2, 3, 4, 5)
	cloned := original.Clone()

	// Get slices from both
	originalSlice := original.ToSlice()
	clonedSlice := cloned.ToSlice()

	// Modify original slice
	originalSlice[0] = 999

	// Cloned should be unaffected
	require.Equal(t, 1, cloned.First())
	require.NotEqual(t, originalSlice, clonedSlice)
}

func TestTupleNCloneDeepIndependence(t *testing.T) {
	// Create tuple with slice of integers
	tn := tuple.NewTupleN(10, 20, 30)
	cloned := tn.Clone()

	// Both should be equal
	require.True(t, tn.Equals(cloned))

	// Get internal slices via ToSlice (which clones)
	original := tn.ToSlice()
	copy := cloned.ToSlice()

	// Modify the slices
	original[0] = 999

	// The tuple values themselves shouldn't be affected
	// because ToSlice() returns a clone
	require.Equal(t, 10, tn.First())
	require.Equal(t, 10, cloned.First())
	require.NotEqual(t, original[0], copy[0])
}
