package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strings"
	"sync"
)

// Error is this framework's error type. It adds on top of the go std lib errors package:
// - A Code field to categorize the error.
// - A Stacktrace that pretty prints with wrapped errors.
// - A Conversion to slog.Value to log the error in a structured way.
// - A Conversion to JSON to log the error in a structured way.
type Error struct {
	Code             string
	Message          string
	shortErrorString string
	wrappedErrs      []error
	mu               sync.Mutex
	hideStack        bool
	stack            _Stack

	codeKey    string
	messageKey string
	stackKey   string
}

// Default keys for the JSON and slog.Value representations of the error.
// The current values are compatible with DataDog.
// Change them if you want to use a different format.
// Only errors that are created after the change will be affected.
var (
	defaultCodeKey    = "kind"
	defaultMessageKey = "message"
	defaultStackKey   = "stack"
)

// SetDefaultCodeKey sets the default key used to represent the error code in the JSON and slog.Value representations of the error.
// Empty value is not allowed
func SetDefaultCodeKey(key string) {
	if key == "" {
		panic("code key cannot be empty")
	}
	defaultCodeKey = key
}

// SetDefaultMessageKey sets the default key used to represent the error message in the JSON and slog.Value representations of the error.
// Empty value means do not include the message in the JSON and slog.Value representations.
func SetDefaultMessageKey(key string) {
	defaultMessageKey = key
}

// SetDefaultStackKey sets the default key used to represent the error stacktrace in the JSON and slog.Value representations of the error.
// Empty value means do not include the stacktrace in the JSON and slog.Value representations.
func SetDefaultStackKey(key string) {
	defaultStackKey = key
}

// Newf creates a new error with the given code and message format/args.
// The error will have a stacktrace attached to it.
// It follows the same rules as fmt.Errorf, where the message is formatted with fmt.Sprintf.
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

		codeKey:    defaultCodeKey,
		messageKey: defaultMessageKey,
		stackKey:   defaultStackKey,
	}
	fwErr.shortErrorString = fmt.Sprintf("{%s} %s", fwErr.Code, fwErr.Message)
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

	return e.buildFullErrorString()
}

func (e *Error) buildFullErrorString() string {
	buf := strings.Builder{}
	buf.WriteString(e.shortErrorString)
	buf.WriteString("\nstacktrace:\n")
	_ = e.WriteStacktrace(&buf)
	return buf.String()
}

// Unwrap returns the errors that have been directly wrapped by err, if any.
func (e *Error) Unwrap() []error {
	return e.wrappedErrs
}

// Stacktrace returns the error stack trace as a string. The output us produced by calling WriteStacktrace.
func (e *Error) Stacktrace() string {
	buf := strings.Builder{}
	_ = e.WriteStacktrace(&buf)
	return buf.String()
}

// WriteStacktrace writes the error stack trace to the writer. It also includes the stack trace of any wrapped errors.
func (e *Error) WriteStacktrace(w io.Writer) error {
	err := e.WriteSelfStacktrace(w)
	if err != nil {
		return err
	}
	return writeWrappedStacktrace(e, w, "[")
}

// SelfStacktrace returns just the error stack trace as a string. The output us produced by calling WriteSelfStacktrace.
func (e *Error) SelfStacktrace() string {
	buf := strings.Builder{}
	_ = e.WriteSelfStacktrace(&buf)
	return buf.String()
}

// WriteSelfStacktrace writes just the error stack trace to the writer.
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

// SetCodeKey sets the key used to represent the error code in the JSON and slog.Value representations of the error.
// Empty value is not allowed
func (e *Error) SetCodeKey(key string) {
	if key == "" {
		panic("code key cannot be empty")
	}
	e.codeKey = key
}

// SetMessageKey sets the key used to represent the error message in the JSON and slog.Value representations of the error.
// Empty value means do not include the message in the JSON and slog.Value representations.
func (e *Error) SetMessageKey(key string) {
	e.messageKey = key
}

// SetStackKey sets the key used to represent the error stacktrace in the JSON and slog.Value representations of the error.
// Empty value means do not include the stacktrace in the JSON and slog.Value representations.
func (e *Error) SetStackKey(key string) {
	e.stackKey = key
}

// LogValue returns a slog.Value that can be used to log the error.
// The default format is compatible with DataDog.
func (e *Error) LogValue() slog.Value {
	items := make([]slog.Attr, 0, 3)
	items = append(items, slog.String(e.codeKey, e.Code))
	if len(e.messageKey) > 0 {
		items = append(items, slog.String(e.messageKey, e.Message))
	}
	if len(e.stackKey) > 0 {
		items = append(items, slog.String(e.stackKey, e.Stacktrace()))
	}
	return slog.GroupValue(items...)
}

// MarshalJSON implements the json.Marshaler interface.
// The default format is compatible with DataDog.
func (e *Error) MarshalJSON() ([]byte, error) {
	mapping := map[string]string{
		e.codeKey: e.Code,
	}
	if len(e.messageKey) > 0 {
		mapping[e.messageKey] = e.Message
	}
	if len(e.stackKey) > 0 {
		mapping[e.stackKey] = e.Stacktrace()
	}
	return json.Marshal(mapping)
}

// Copy creates a deep copy of the error.
func (e *Error) Copy() *Error {
	return &Error{
		Code:        e.Code,
		Message:     e.Message,
		wrappedErrs: e.wrappedErrs,
		stack:       e.stack,

		codeKey:    e.codeKey,
		messageKey: e.messageKey,
		stackKey:   e.stackKey,
	}
}

// NewUnknownf creates a new error with the ErrCodeUnknown code and the given message format/args.
func NewUnknownf(format string, args ...any) *Error {
	return Newf(ErrCodeUnknown, format, args...)
}

// IsCode returns true if the error, or any wrapped error, is of type Error and has the given code.
func IsCode(err error, code string) bool {
	var fwErr *Error
	if As(err, &fwErr) && fwErr.Code == code {
		return true
	}

	// As function will only return the first match, so we need to manually unwrap the error chain.
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

// AsCode returns true if the error, or any wrapped error, is of type Error and has the given code.
// The found error is assigned to target.
func AsCode(err error, target **Error, code string) bool {
	if As(err, target) && (*target).Code == code {
		return true
	}

	// As function will only return the first match, so we need to manually unwrap the error chain.
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

// UnwrapMulti returns all the errors that have been directly wrapped by err, if any.
// This is similar to Unwrap, but it also unwraps errors that implement:
//
//	interface { Unwrap() []error} }
func UnwrapMulti(err error) []error {
	switch e := err.(type) {
	case interface{ Unwrap() error }:
		if w := e.Unwrap(); w != nil {
			return []error{w}
		}
	case interface{ Unwrap() []error }:
		return e.Unwrap()
	}
	return nil
}

//#region copy from errors.go

// ErrUnsupported is a copy of the errors.ErrUnsupported variable from the go std core.
var ErrUnsupported = errors.ErrUnsupported

// Join is a copy of the errors.Join function from the go std core.
var Join = errors.Join

// Unwrap is a copy of the errors.Unwrap function from the go std core.
var Unwrap = errors.Unwrap

// As is a copy of the errors.As function from the go std core.
var As = errors.As

// Is is a copy of the errors.Is function from the go std core.
var Is = errors.Is

//#endregion
