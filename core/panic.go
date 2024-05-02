package core

import (
	"os"

	"github.com/southernlabs-io/go-fw/errors"
)

// DeferredPanicToError is a panic handler that sets the given err to a new error with the given format and args.
// If err is not nil, it will be wrapped by the new error as a hidden error.
func DeferredPanicToError(err *error, format string, args ...any) {
	if r := recover(); r != nil {
		if err != nil && *err != nil {
			*err = errors.Newf(errors.ErrCodePanic, format+", panic: %v, hidden error: %w", append(args, r, *err)...)
		} else {
			*err = errors.Newf(errors.ErrCodePanic, format+", panic: %v", append(args, r)...)
		}
	}
}

// DeferredPanicToLogAndExit is a panic handler that logs the panic and exits with code 2.
func DeferredPanicToLogAndExit() {
	if errAny := recover(); errAny != nil {
		if err, ok := errAny.(*errors.Error); ok {
			GetLogger().Errorf("panic: %v", err)
		} else {
			GetLogger().Errorf("panic: %v", errors.Newf(errors.ErrCodePanic, "%v", errAny))
		}
		os.Exit(2) // Replicate a regular panic exit value
	}
}
