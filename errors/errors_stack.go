package errors

import (
	"io"
	"reflect"
	"runtime"
	"strconv"
	"sync"
)

type _Stack []uintptr

var packageFuncPrefix = sync.OnceValue(func() string {
	return reflect.TypeOf(Error{}).PkgPath() + "."
})

func writeWrappedStacktrace(err error, w io.Writer, indent string) error {
	if indent != "[" {
		indent += "."
	}
	for i, wErr := range UnwrapMulti(err) {
		curIndent := indent + strconv.Itoa(i+1)
		_, err := w.Write([]byte("\n" + curIndent))
		if err != nil {
			return err
		}
		if fwErr, is := wErr.(*Error); is {
			_, err := w.Write([]byte("] wrapped stacktrace:\n"))
			if err != nil {
				return err
			}
			err = fwErr.WriteSelfStacktrace(w)
			if err != nil {
				return err
			}
		} else {
			_, err := w.Write([]byte("] wrapped stacktrace not available for error type: "))
			if err != nil {
				return err
			}
			_, err = w.Write([]byte(reflect.TypeOf(wErr).String()))
			if err != nil {
				return err
			}
		}
		err = writeWrappedStacktrace(wErr, w, curIndent)
		if err != nil {
			return err
		}
	}

	return nil
}

func currStack() _Stack {
	stackPtrs := make(_Stack, 20)
	count := runtime.Callers(3, stackPtrs)
	stack := stackPtrs[0:count]
	return stack
}
