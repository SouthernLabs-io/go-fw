package resterrors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
)

// ErrorCodeHTTPMapperFunc is a function that maps an error to an HTTP status code.
// It should return 0 if it cannot map the error, in which case the default mapping will be used.
type ErrorCodeHTTPMapperFunc func(ctx context.Context, err error) int

var ErrorCodeMapper ErrorCodeHTTPMapperFunc

func ErrorHandler(w http.ResponseWriter, r *http.Request, err *errors.Error) {
	ctx := r.Context()
	httpCode := mapErrorToHTTPCode(ctx, err)
	jsonStr, jsonErr := json.Marshal(err)
	if jsonErr != nil {
		log.GetLoggerFromCtx(ctx).Errorf("failed to marshal error response, sending text as is: %v", jsonErr)
		http.Error(w, err.Error(), httpCode)
	} else {
		// This is the same implementation as in http.Error as of go1.25, only difference is we are using application/json
		headers := w.Header()
		headers.Set("Content-Type", "application/json")
		headers.Del("Content-Length") // let net/http set it
		headers.Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(httpCode)
		fmt.Fprintln(w, string(jsonStr))
	}
}

func mapErrorToHTTPCode(ctx context.Context, err *errors.Error) int {
	if ErrorCodeMapper != nil {
		if code := ErrorCodeMapper(ctx, err); code != 0 {
			log.GetLoggerFromCtx(ctx).Debugf("Using custom error code mapper for error: %v", err)
			return code
		}
		log.GetLoggerFromCtx(ctx).Debugf("Custom error code mapper returned 0 for error: %v, using default mapping", err)
	}

	// Default mapping
	switch err.Code {
	case errors.ErrCodeNotAuthenticated:
		return http.StatusUnauthorized
	case errors.ErrCodeNotAllowed:
		return http.StatusForbidden
	case errors.ErrCodeNotFound:
		return http.StatusNotFound
	case errors.ErrCodeConflict:
		return http.StatusConflict
	case errors.ErrCodeBadArgument:
		return http.StatusBadRequest
	case errors.ErrCodeValidationFailed:
		return http.StatusUnprocessableEntity
	case errors.ErrCodeBadState:
		return http.StatusInternalServerError
	case errors.ErrCodeUnknown:
		return http.StatusInternalServerError
	default:
		log.GetLoggerFromCtx(ctx).Debugf("No specific HTTP mapping for error code: %s, using 500", err.Code)
		return http.StatusInternalServerError
	}
}
