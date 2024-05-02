package predicates_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/ptr"

	"github.com/southernlabs-io/go-fw/functional/predicates"
)

type Interface interface {
}

func TestNil(t *testing.T) {
	require.False(t, predicates.Nil(""))
	require.False(t, predicates.Nil("hi"))
	require.True(t, predicates.Nil[any](nil))
	require.True(t, predicates.Nil((func())(nil)))
	require.False(t, predicates.Nil(ptr.ToPtr("hi")))
	require.False(t, predicates.Nil(ptr.ToPtr("")))

	var aSlice []any
	require.True(t, predicates.Nil(aSlice))
	require.True(t, predicates.Nil(aSlice))

	aSlice = make([]any, 0)
	require.False(t, predicates.Nil(aSlice))
	require.False(t, predicates.Nil(aSlice))

	var aMap map[any]any
	require.True(t, predicates.Nil(aMap))
	aMap = make(map[any]any, 0)
	require.False(t, predicates.Nil(aMap))
	require.False(t, predicates.Nil(aMap))

	var aInterface Interface
	require.True(t, predicates.Nil(aInterface))
	require.True(t, predicates.Nil(aInterface))

	aInterface = "hi"
	require.False(t, predicates.Nil(aInterface))
	require.False(t, predicates.Nil(aInterface))

	aInterface = (*string)(nil)
	require.True(t, predicates.Nil(aInterface))
	require.True(t, predicates.Nil(aInterface))
}

func TestEmpty(t *testing.T) {
	require.True(t, predicates.Empty(""))
	require.False(t, predicates.Empty(" "))

	require.True(t, predicates.Empty((func())(nil)))
	require.False(t, predicates.Empty(func() {}))

	var aSlice []any
	require.True(t, predicates.Empty(aSlice))

	aSlice = make([]any, 0)
	require.True(t, predicates.Empty(aSlice))

	aSlice = append(aSlice, "hi")
	require.False(t, predicates.Empty(aSlice))

	require.True(t, predicates.Empty[any](nil))
	var strPtr *string
	require.True(t, predicates.Empty(strPtr))
	strPtr = ptr.ToPtr("")
	require.True(t, predicates.Empty(strPtr))

}

func TestEmptyI(t *testing.T) {
	require.True(t, predicates.EmptyI(0, ""))
	require.False(t, predicates.EmptyI(0, " "))

	require.True(t, predicates.EmptyI(0, (func())(nil)))
	require.False(t, predicates.EmptyI(0, func() {}))

	var aSlice []any
	require.True(t, predicates.EmptyI(0, aSlice))

	aSlice = make([]any, 0)
	require.True(t, predicates.EmptyI(0, aSlice))

	aSlice = append(aSlice, "hi")
	require.False(t, predicates.EmptyI(0, aSlice))
}

func TestNot(t *testing.T) {
	require.False(t, predicates.Not(predicates.Empty[string])(""))
	require.True(t, predicates.Not(predicates.Empty[string])(" "))
}

func TestNotI(t *testing.T) {
	require.False(t, predicates.NotI(predicates.EmptyI[string])(0, ""))
	require.True(t, predicates.NotI(predicates.EmptyI[string])(0, " "))
}
