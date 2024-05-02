package maps_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/functional/maps"
)

func TestToSlice(t *testing.T) {
	m := map[int]string{1: "a", 2: "b", 3: "c", 10: "j"}
	entries := maps.ToSlice(m)
	require.EqualValues(t, 4, len(entries))
	slices.SortFunc(entries, func(a, b maps.Entry[int, string]) int {
		return a.Key - b.Key
	})
	require.EqualValues(t, 1, entries[0].Key)
	require.EqualValues(t, "a", entries[0].Value)
	require.EqualValues(t, 2, entries[1].Key)
	require.EqualValues(t, "b", entries[1].Value)
	require.EqualValues(t, 3, entries[2].Key)
	require.EqualValues(t, "c", entries[2].Value)
	require.EqualValues(t, 10, entries[3].Key)
	require.EqualValues(t, "j", entries[3].Value)
}

func TestFilter(t *testing.T) {
	m := map[int]string{1: "a", 2: "b", 3: "c", 10: "j"}
	filtered := maps.Filter(m, func(entry maps.Entry[int, string]) bool {
		return entry.Key%2 == 0
	})
	require.EqualValues(t, 2, len(filtered))
	require.EqualValues(t, "b", filtered[2])
	require.EqualValues(t, "j", filtered[10])
}

func TestMap(t *testing.T) {
	m := map[int]string{1: "a", 2: "b", 3: "c", 10: "j"}
	mapped := maps.Map(m, func(entry maps.Entry[int, string]) maps.Entry[int64, rune] {
		return maps.Entry[int64, rune]{Key: int64(entry.Key * 2), Value: rune(entry.Value[0])}
	})
	require.EqualValues(t, 4, len(mapped))
	require.EqualValues(t, 'a', mapped[2])
	require.EqualValues(t, 'b', mapped[4])
	require.EqualValues(t, 'c', mapped[6])
	require.EqualValues(t, 'j', mapped[20])
}

func TestMapE(t *testing.T) {
	m := map[int]string{1: "a", 2: "b", 3: "c", 10: "j"}
	mapped, err := maps.MapE(m, func(entry maps.Entry[int, string]) (maps.Entry[int64, rune], error) {
		return maps.Entry[int64, rune]{Key: int64(entry.Key * 2), Value: rune(entry.Value[0])}, nil
	})
	require.NoError(t, err)
	require.EqualValues(t, 4, len(mapped))
	require.EqualValues(t, 'a', mapped[2])
	require.EqualValues(t, 'b', mapped[4])
	require.EqualValues(t, 'c', mapped[6])
	require.EqualValues(t, 'j', mapped[20])

	mapped, err = maps.MapE(m, func(entry maps.Entry[int, string]) (maps.Entry[int64, rune], error) {
		return maps.Entry[int64, rune]{}, errors.Newf("MAP_ERROR", "test")
	})
	require.Error(t, err)
	require.True(t, errors.IsCode(err, "MAP_ERROR"))
	require.Nil(t, mapped)
}

func TestFromEntries(t *testing.T) {
	entries := []maps.Entry[int, string]{
		{Key: 1, Value: "a"},
		{Key: 2, Value: "b"},
		{Key: 3, Value: "c"},
		{Key: 10, Value: "j"},
	}
	m := maps.FromEntries(entries)
	require.EqualValues(t, 4, len(m))
	require.EqualValues(t, "a", m[1])
	require.EqualValues(t, "b", m[2])
	require.EqualValues(t, "c", m[3])
	require.EqualValues(t, "j", m[10])
}
