// Code generated by "stringer -type=ConcurrencyMode"; DO NOT EDIT.

package worker

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ConcurrencyModeMulti-0]
	_ = x[ConcurrencyModeSingle-1]
}

const _ConcurrencyMode_name = "ConcurrencyModeMultiConcurrencyModeSingle"

var _ConcurrencyMode_index = [...]uint8{0, 20, 41}

func (i ConcurrencyMode) String() string {
	if i < 0 || i >= ConcurrencyMode(len(_ConcurrencyMode_index)-1) {
		return "ConcurrencyMode(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ConcurrencyMode_name[_ConcurrencyMode_index[i]:_ConcurrencyMode_index[i+1]]
}