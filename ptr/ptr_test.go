package ptr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatPtr(t *testing.T) {
	var strPtr *string
	require.Equal(t, "<nil>", FormatPtr(strPtr))
	var str = "hi"
	require.Equal(t, "hi", FormatPtr(&str))
}

func TestToPtr(t *testing.T) {
	var str = "hi"
	require.Equal(t, &str, ToPtr(str))

	var intVal = 123
	require.Equal(t, &intVal, ToPtr(intVal))

	var m map[string]int
	require.Equal(t, map[string]int(nil), m)
	require.NotNil(t, ToPtr(m))
}

func TestToPtrIfNotNil(t *testing.T) {
	var m map[string]int
	require.Nil(t, ToPtrIfNotNil(m))

	m = map[string]int{"a": 1}
	ptr := ToPtrIfNotNil(m)
	require.NotNil(t, ptr)
	require.Equal(t, m, *ptr)

	var ch chan int
	require.Nil(t, ToPtrIfNotNil(ch))

	ch = make(chan int)
	ptrCh := ToPtrIfNotNil(ch)
	require.NotNil(t, ptrCh)
	require.Equal(t, ch, *ptrCh)
}

func TestToPtrIfNotZero(t *testing.T) {
	var str string
	require.Nil(t, ToPtrIfNotZero(str))

	str = "hi"
	ptr := ToPtrIfNotZero(str)
	require.NotNil(t, ptr)
	require.Equal(t, str, *ptr)

	var intVal int
	require.Nil(t, ToPtrIfNotZero(intVal))

	intVal = 123
	ptrInt := ToPtrIfNotZero(intVal)
	require.NotNil(t, ptrInt)
	require.Equal(t, intVal, *ptrInt)

	var structVal struct{ field string }
	require.Nil(t, ToPtrIfNotZero(structVal))

	structVal = struct{ field string }{field: "value"}
	ptrStruct := ToPtrIfNotZero(structVal)
	require.NotNil(t, ptrStruct)
	require.Equal(t, structVal, *ptrStruct)
}

func TestToValue(t *testing.T) {
	var strPtr *string
	require.Equal(t, "", ToValue(strPtr))

	var str = "hi"
	require.Equal(t, "hi", ToValue(&str))

	var intVal = 123
	require.Equal(t, 123, ToValue(&intVal))
}
