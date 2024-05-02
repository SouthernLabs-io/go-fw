package slices_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/functional/predicates"
	"github.com/southernlabs-io/go-fw/functional/slices"
	"github.com/southernlabs-io/go-fw/ptr"
)

func TestFilter(t *testing.T) {
	var aZeroMap map[any]any
	var aZeroSlice []any
	aSlice := []any{
		"hi",
		nil,
		(*string)(nil),
		ptr.ToPtr("hi"),
		(func())(nil),
		func() {},
		aZeroMap,
		map[string]string{},
		aZeroSlice,
		[]any{},
	}

	notNilSlice := slices.Filter(aSlice, predicates.Not(predicates.Nil[any]))
	require.Equal(t, 5, len(notNilSlice))
	require.Equal(t, 5, len(slices.Filter(aSlice, predicates.Nil[any])))
	require.Equal(t, 2, len(slices.Filter(
		notNilSlice,
		predicates.Empty[any],
	)))
}

func TestFilterI(t *testing.T) {
	var aZeroMap map[any]any
	var aZeroSlice []any
	aSlice := []any{
		"hi",
		nil,
		(*string)(nil),
		ptr.ToPtr("hi"),
		(func())(nil),
		func() {},
		aZeroMap,
		map[string]string{},
		aZeroSlice,
		[]any{},
	}

	notNilSlice := slices.FilterI(aSlice, predicates.NotI(predicates.NilI[any]))
	require.Equal(t, 5, len(notNilSlice))
	require.Equal(t, 5, len(slices.FilterI(aSlice, predicates.NilI[any])))
	require.Equal(t, 2, len(slices.FilterI(
		notNilSlice,
		predicates.EmptyI[any],
	)))
}

func TestMap(t *testing.T) {
	aSliceOfNames := []string{"John", "Peter", "Mary"}
	expected := []string{"4a6f686e", "5065746572", "4d617279"}

	result := slices.Map(aSliceOfNames, func(e string) string {
		return fmt.Sprintf("%x", e)
	})
	require.NotNil(t, result)
	require.Equal(t, len(aSliceOfNames), len(result))
	require.EqualValues(t, expected, result)
}

func TestMapI(t *testing.T) {
	aSliceOfNames := []string{"John", "Peter", "Mary"}
	expected := []string{"0:John", "1:Peter", "2:Mary"}

	result := slices.MapI(aSliceOfNames, func(i int, e string) string {
		return fmt.Sprintf("%x:%s", i, e)
	})
	require.NotNil(t, result)
	require.Equal(t, len(aSliceOfNames), len(result))
	require.EqualValues(t, expected, result)
}

func TestMapIE(t *testing.T) {
	aSliceOfNames := []string{"John", "Peter", "Mary"}
	expected := []string{"0:John", "1:Peter", "2:Mary"}

	result, err := slices.MapIE(aSliceOfNames, func(i int, e string) (string, error) {
		return fmt.Sprintf("%x:%s", i, e), nil
	})
	require.Nil(t, err)
	require.NotNil(t, result)
	require.Equal(t, len(aSliceOfNames), len(result))
	require.EqualValues(t, expected, result)

	result, err = slices.MapIE(aSliceOfNames, func(i int, e string) (string, error) {
		return "", errors.NewUnknownf("mapping failure")
	})
	require.Nil(t, result)
	require.NotNil(t, err)
	var fwErr *errors.Error
	require.ErrorAs(t, err, &fwErr)
	require.EqualValues(t, errors.ErrCodeUnknown, fwErr.Code)
	require.EqualValues(t, "mapping failure", fwErr.Message)
}

func TestFindMap(t *testing.T) {
	aSliceOfNames := []string{"John", "Peter", "Mary"}

	find := "Peter"
	value, idx := slices.FindMap(aSliceOfNames, func(e string) (string, bool) {
		return e, e == find
	})
	require.Equal(t, find, value)
	require.Equal(t, 1, idx)

	value, idx = slices.FindMap(aSliceOfNames, func(e string) (string, bool) {
		return e, false
	})
	require.Equal(t, "", value)
	require.Equal(t, -1, idx)
}

func TestFindMapI(t *testing.T) {
	aSliceOfNames := []string{"John", "Peter", "Mary"}

	find := "Peter"
	value, idx := slices.FindMapI(aSliceOfNames, func(i int, e string) (string, bool) {
		return e, e == find
	})
	require.Equal(t, find, value)
	require.Equal(t, 1, idx)

	value, idx = slices.FindMapI(aSliceOfNames, func(i int, e string) (string, bool) {
		return e, false
	})
	require.Equal(t, "", value)
	require.Equal(t, -1, idx)
}

func TestFindMapIE(t *testing.T) {
	aSliceOfNames := []string{"John", "Peter", "Mary"}

	find := "Peter"
	value, idx, err := slices.FindMapIE(aSliceOfNames, func(i int, e string) (string, bool, error) {
		return e, e == find, nil
	})
	require.Nil(t, err)
	require.Equal(t, find, value)
	require.Equal(t, 1, idx)

	value, idx, err = slices.FindMapIE(aSliceOfNames, func(i int, e string) (string, bool, error) {
		return e, false, nil
	})
	require.Nil(t, err)
	require.Empty(t, value)
	require.Equal(t, -1, idx)

	value, idx, err = slices.FindMapIE(aSliceOfNames, func(i int, e string) (string, bool, error) {
		return e, false, errors.NewUnknownf("mapping failure")
	})
	require.Empty(t, value)
	require.EqualValues(t, idx, -1)
	require.NotNil(t, err)
	var fwErr *errors.Error
	require.ErrorAs(t, err, &fwErr)
	require.EqualValues(t, errors.ErrCodeUnknown, fwErr.Code)
	require.EqualValues(t, "mapping failure", fwErr.Message)
}

func TestFindLastMap(t *testing.T) {
	aSliceOfNames := []string{"John", "Peter", "Mary", "Peter"}

	find := "Peter"
	value, idx := slices.FindLastMap(aSliceOfNames, func(e string) (string, bool) {
		return e, e == find
	})
	require.Equal(t, find, value)
	require.Equal(t, 3, idx)

	value, idx = slices.FindLastMap(aSliceOfNames, func(e string) (string, bool) {
		return e, false
	})
	require.Equal(t, "", value)
	require.Equal(t, -1, idx)
}

func TestFindLastMapI(t *testing.T) {
	aSliceOfNames := []string{"John", "Peter", "Mary", "Peter"}

	find := "Peter"
	value, idx := slices.FindLastMapI(aSliceOfNames, func(i int, e string) (string, bool) {
		return e, e == find
	})
	require.Equal(t, find, value)
	require.Equal(t, 3, idx)

	value, idx = slices.FindLastMapI(aSliceOfNames, func(i int, e string) (string, bool) {
		return e, false
	})
	require.Equal(t, "", value)
	require.Equal(t, -1, idx)
}

func TestFindLastMapIE(t *testing.T) {
	aSliceOfNames := []string{"John", "Peter", "Mary", "Peter"}

	find := "Peter"
	value, idx, err := slices.FindLastMapIE(aSliceOfNames, func(i int, e string) (string, bool, error) {
		return e, e == find, nil
	})
	require.Nil(t, err)
	require.Equal(t, find, value)
	require.Equal(t, 3, idx)

	value, idx, err = slices.FindLastMapIE(aSliceOfNames, func(i int, e string) (string, bool, error) {
		return e, false, nil
	})
	require.Nil(t, err)
	require.Equal(t, "", value)
	require.Equal(t, -1, idx)

	value, idx, err = slices.FindLastMapIE(aSliceOfNames, func(i int, e string) (string, bool, error) {
		return e, false, errors.NewUnknownf("mapping failure")
	})
	require.Empty(t, value)
	require.EqualValues(t, idx, -1)
	require.NotNil(t, err)
	var fwErr *errors.Error
	require.ErrorAs(t, err, &fwErr)
	require.EqualValues(t, errors.ErrCodeUnknown, fwErr.Code)
	require.EqualValues(t, "mapping failure", fwErr.Message)
}

func TestFlatMap(t *testing.T) {
	aSliceOfNames := [][]string{{"John", "Peter"}, {"Mary"}, {}}
	expected := []string{"John", "Peter", "Mary"}

	result := slices.FlatMap(aSliceOfNames, func(e []string) []string {
		return e
	})
	require.NotNil(t, result)
	require.Equal(t, len(aSliceOfNames), len(result))
	require.EqualValues(t, expected, result)
}

func TestFlatMapI(t *testing.T) {
	aSliceOfNames := [][]string{{"John", "Peter"}, {"Mary"}, {}}
	expected := []string{"0:John", "0:Peter", "1:Mary"}

	result := slices.FlatMapI(aSliceOfNames, func(i int, e []string) []string {
		return slices.Map(e, func(s string) string {
			return fmt.Sprintf("%d:%s", i, s)
		})
	})
	require.NotNil(t, result)
	require.Equal(t, len(aSliceOfNames), len(result))
	require.EqualValues(t, expected, result)
}

func TestFlatMapIE(t *testing.T) {
	aSliceOfNames := [][]string{{"John", "Peter"}, {"Mary"}, {}}
	expected := []string{"0:0:John", "0:1:Peter", "1:0:Mary"}

	result, err := slices.FlatMapIE(aSliceOfNames, func(i int, e []string) ([]string, error) {
		return slices.MapIE(e, func(j int, s string) (string, error) {
			return fmt.Sprintf("%d:%d:%s", i, j, s), nil
		})
	})
	require.Nil(t, err)
	require.NotNil(t, result)
	require.Equal(t, len(aSliceOfNames), len(result))
	require.EqualValues(t, expected, result)

	result, err = slices.FlatMapIE(aSliceOfNames, func(i int, e []string) ([]string, error) {
		return nil, errors.Newf("MAP_ERROR", "mapping failure")
	})
	require.Nil(t, result)
	require.NotNil(t, err)
	require.True(t, errors.IsCode(err, "MAP_ERROR"))
}
