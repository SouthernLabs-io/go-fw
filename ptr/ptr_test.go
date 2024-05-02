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
}

func TestToValue(t *testing.T) {
	var strPtr *string
	require.Equal(t, "", ToValue(strPtr))

	var str = "hi"
	require.Equal(t, "hi", ToValue(&str))

	var intVal = 123
	require.Equal(t, 123, ToValue(&intVal))
}
