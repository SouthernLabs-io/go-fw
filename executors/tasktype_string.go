// Code generated by "stringer -type=_TaskType"; DO NOT EDIT.

package executors

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[_TaskTypeOnce-0]
	_ = x[_TaskTypeFixedDelay-1]
	_ = x[_TaskTypeFixedRate-2]
}

const _TaskType_name = "TaskTypeOnceTaskTypeFixedDelayTaskTypeFixedRate"

var _TaskType_index = [...]uint8{0, 12, 30, 47}

func (i _TaskType) String() string {
	if i < 0 || i >= _TaskType(len(_TaskType_index)-1) {
		return "_TaskType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _TaskType_name[_TaskType_index[i]:_TaskType_index[i+1]]
}
