package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var ErrUnsupported = errors.ErrUnsupported

type _Stack []uintptr

type Error struct {
	Code             string
	Message          string
	shortErrorString string
	fullErrorString  func() string
	wrappedErrs      []error
	mu               sync.Mutex
	hideStack        bool
	stack            _Stack
}

func Newf(code string, format string, args ...any) *Error {
	// Hide the stacktrace from the Error() function for errors that are going to be wrapped
	for _, errArg := range args {
		if fwErr, ok := errArg.(*Error); ok {
			fwErr.mu.Lock()
			fwErr.hideStack = true
		}
	}
	defer func() {
		// Show the stacktrace from the Error() function
		for _, errArg := range args {
			if fwErr, ok := errArg.(*Error); ok {
				fwErr.hideStack = false
				fwErr.mu.Unlock()
			}
		}
	}()
	err := fmt.Errorf(format, args...)

	var wrappedErrs []error
	switch x := err.(type) {
	case interface{ Unwrap() error }:
		wrappedErrs = append(wrappedErrs, x.Unwrap())
	case interface{ Unwrap() []error }:
		wrappedErrs = x.Unwrap()
	}

	fwErr := &Error{
		Code:        code,
		Message:     err.Error(),
		wrappedErrs: wrappedErrs,
		stack:       currStack(),
	}
	fwErr.shortErrorString = fmt.Sprintf("{%s} %s", fwErr.Code, fwErr.Message)
	fwErr.fullErrorString = fwErr.buildFullErrorString
	return fwErr
}

func (e *Error) Error() string {
	if e.hideStack {
		return e.shortErrorString
	}

	// Don't print the stacktrace if the error is being wrapped by fmt.Errorf
	stack := currStack()
	for _, pc := range stack {
		f := runtime.FuncForPC(pc)
		if f != nil && f.Name() == "fmt.Errorf" {
			return e.shortErrorString
		}
	}

	return e.fullErrorString()
}

func (e *Error) buildFullErrorString() string {
	buf := strings.Builder{}
	buf.WriteString(e.shortErrorString)
	buf.WriteString("\nstacktrace:\n")
	_ = e.WriteStacktrace(&buf)
	return buf.String()
}

func (e *Error) Unwrap() []error {
	return e.wrappedErrs
}

var packageFuncPrefix = sync.OnceValue(func() string {
	return reflect.TypeOf(Error{}).PkgPath() + "."
})

func (e *Error) Stacktrace() string {
	buf := strings.Builder{}
	_ = e.WriteStacktrace(&buf)
	return buf.String()
}

func (e *Error) WriteStacktrace(w io.Writer) error {
	err := e.WriteSelfStacktrace(w)
	if err != nil {
		return err
	}
	return writeWrappedStacktrace(e, w, "[")
}

func (e *Error) SelfStacktrace() string {
	buf := strings.Builder{}
	_ = e.WriteSelfStacktrace(&buf)
	return buf.String()
}

func (e *Error) WriteSelfStacktrace(w io.Writer) error {
	if len(e.stack) == 0 {
		return nil
	}
	frames := runtime.CallersFrames(e.stack)
	prefix := packageFuncPrefix()
	for {
		f, more := frames.Next()
		if f.Function != "" && !strings.HasPrefix(f.Function, prefix) {
			_, err := w.Write([]byte(fmt.Sprintf("%s\n\t%s:%d", f.Function, f.File, f.Line)))
			if err != nil {
				return err
			}
			if more {
				_, err := w.Write([]byte("\n"))
				if err != nil {
					return err
				}
			}
		}
		if !more {
			break
		}
	}
	return nil
}

func (e *Error) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("kind", e.Code),
		slog.String("message", e.Message),
		slog.String("stack", e.Stacktrace()),
	)
}

func (e *Error) MarshalJSON() ([]byte, error) {
	mapping := map[string]string{
		"kind":    e.Code,
		"message": e.Message,
		"stack":   e.Stacktrace(),
	}
	return json.Marshal(mapping)
}

func NewUnknownf(format string, args ...any) *Error {
	return Newf(ErrCodeUnknown, format, args...)
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func IsCode(err error, code string) bool {
	// Fast path
	var fxErr *Error
	if As(err, &fxErr) && fxErr.Code == code {
		return true
	}

	// Slow path, manual unwrapping to check the code on all the errors in the chain
	switch e := err.(type) {
	case interface{ Unwrap() error }:
		return IsCode(e.Unwrap(), code)
	case interface{ Unwrap() []error }:
		for _, wrappedErr := range e.Unwrap() {
			if IsCode(wrappedErr, code) {
				return true
			}
		}
	}

	return false
}

func As(err error, target any) bool {
	// Is it ok to pass the target as is
	return errors.As(err, target)
}

func AsCode(err error, target **Error, code string) bool {
	// Fast path
	if As(err, target) && (*target).Code == code {
		return true
	}

	// Slow path, manual unwrapping to check the code on all the errors in the chain
	switch e := err.(type) {
	case interface{ Unwrap() error }:
		return AsCode(e.Unwrap(), target, code)
	case interface{ Unwrap() []error }:
		for _, wrappedErr := range e.Unwrap() {
			if AsCode(wrappedErr, target, code) {
				return true
			}
		}
	}

	return false
}

func Unwrap(err error) []error {
	switch e := err.(type) {
	case interface{ Unwrap() error }:
		if w := e.Unwrap(); w != nil {
			return []error{w}
		}
		return nil
	case interface{ Unwrap() []error }:
		return e.Unwrap()
	default:
		return nil
	}
}

func Join(errs ...error) error {
	return errors.Join(errs...)
}

func writeWrappedStacktrace(err error, w io.Writer, indent string) error {
	if indent != "[" {
		indent += "."
	}
	for i, wErr := range Unwrap(err) {
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
