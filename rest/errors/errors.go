package resterrors

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

	// Inject error field into logger context for 5xx and non-401/403 4xx errors
	if !(httpCode == http.StatusUnauthorized || httpCode == http.StatusForbidden) {
		if httpCode >= 400 {
			ctx = log.CtxAppendLoggerAttrs(ctx, slog.Any("error", err))
		}
	}

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

	bestStatus := http.StatusInternalServerError
	bestPriority := 5 // 1 is highest priority

	// Default mapping using priority list outside-in
	currError := err
	for currError != nil {
		status, priority := getHTTPStatusAndPriority(currError.Code)
		if priority < bestPriority {
			bestStatus = status
			bestPriority = priority
		}
		currError = currError.FWCause()
	}

	if bestStatus == http.StatusInternalServerError {
		log.GetLoggerFromCtx(ctx).Debugf("No specific HTTP mapping for error code chain starting with: %s, using 500", err.Code)
	}
	return bestStatus
}

func getHTTPStatusAndPriority(code string) (int, int) {
	switch code {
	case errors.ErrCodeNotAuthenticated:
		return http.StatusUnauthorized, 1
	case errors.ErrCodeNotAllowed:
		return http.StatusForbidden, 2
	case errors.ErrCodeNotFound:
		return http.StatusNotFound, 3
	case errors.ErrCodeConflict:
		return http.StatusConflict, 3
	case errors.ErrCodeBadArgument:
		return http.StatusBadRequest, 3
	case errors.ErrCodeValidationFailed:
		return http.StatusUnprocessableEntity, 3
	case errors.ErrCodeBadState:
		return http.StatusInternalServerError, 4
	case errors.ErrCodeUnknown:
		return http.StatusInternalServerError, 4
	default:
		return http.StatusInternalServerError, 4
	}
}
