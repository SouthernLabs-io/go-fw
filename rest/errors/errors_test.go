package resterrors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/southernlabs-io/go-fw/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapErrorToHTTPCode_Priority(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		err      *errors.Error
		expected int
	}{
		{
			name:     "Single specific error",
			err:      errors.NewNotFoundf("user not found"),
			expected: http.StatusNotFound,
		},
		{
			name:     "Single unknown error",
			err:      errors.NewUnknownf("database down"),
			expected: http.StatusInternalServerError,
		},
		{
			name:     "404 wrapped in 500",
			err:      errors.NewUnknownf("query failed: %w", errors.NewNotFoundf("item missing")),
			expected: http.StatusNotFound, // 3 > 4 priority
		},
		{
			name:     "500 wrapped in 404",
			err:      errors.NewNotFoundf("cannot find: %w", errors.NewUnknownf("db error")),
			expected: http.StatusNotFound, // 3 > 4 priority
		},
		{
			name:     "403 wrapped in 404",
			err:      errors.NewNotFoundf("not found: %w", errors.NewNotAllowedf("no permission")),
			expected: http.StatusForbidden, // 2 > 3 priority
		},
		{
			name:     "404 wrapped in 403",
			err:      errors.NewNotAllowedf("no permission: %w", errors.NewNotFoundf("not found")),
			expected: http.StatusForbidden, // 2 > 3 priority
		},
		{
			name:     "404 wrapped in 409 (same priority, outer wins)",
			err:      errors.NewConflictf("conflict: %w", errors.NewNotFoundf("not found")),
			expected: http.StatusConflict, // Same priority 3, outer Conflict wins
		},
		{
			name:     "409 wrapped in 404 (same priority, outer wins)",
			err:      errors.NewNotFoundf("not found: %w", errors.NewConflictf("conflict")),
			expected: http.StatusNotFound, // Same priority 3, outer NotFound wins
		},
		{
			name: "Complex chain: Unknown -> 404 -> 403 -> Unknown",
			err: errors.NewUnknownf("layer 1: %w",
				errors.NewNotFoundf("layer 2: %w",
					errors.NewNotAllowedf("layer 3: %w",
						errors.NewUnknownf("layer 4"),
					),
				),
			),
			expected: http.StatusForbidden, // 403 (priority 2) is the highest
		},
		{
			name: "Complex chain: 500 -> 404 -> 401 -> 403",
			err: errors.NewUnknownf("layer 1: %w",
				errors.NewNotFoundf("layer 2: %w",
					errors.NewNotAuthenticatedf("layer 3: %w",
						errors.NewNotAllowedf("layer 4"),
					),
				),
			),
			expected: http.StatusUnauthorized, // 401 (priority 1) is the highest
		},
		{
			name: "All Priority 3 codes (Validation, Argument, Conflict)",
			err: errors.NewValidationFailedf("invalid param: %w",
				errors.NewBadArgumentf("bad format: %w",
					errors.NewConflictf("db conflict"),
				),
			),
			expected: http.StatusUnprocessableEntity, // ValidationFailed is outer, all are priority 3
		},
		{
			name: "BadState and Unknown chain",
			err: errors.Newf(errors.ErrCodeBadState, "invalid state: %w",
				errors.NewUnknownf("layer 2"),
			),
			expected: http.StatusInternalServerError,
		},
		{
			name:     "Canceled maps to 499",
			err:      errors.NewCanceledf("request canceled"),
			expected: StatusClientClosed,
		},
		{
			name:     "Canceled wrapped in Unknown maps to 499 (higher priority)",
			err:      errors.NewUnknownf("response error: %w", errors.NewCanceledf("request canceled")),
			expected: StatusClientClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := mapErrorToHTTPCode(ctx, tt.err)
			assert.Equal(t, tt.expected, actual)
		})
	}

	t.Run("Custom ErrorCodeMapper successful mapping", func(t *testing.T) {
		defer func() { ErrorCodeMapper = nil }() // Reset
		ErrorCodeMapper = func(ctx context.Context, err error) int {
			if fwErr, ok := err.(*errors.Error); ok && fwErr.Code == errors.ErrCodeUnknown {
				return http.StatusTeapot
			}
			return 0
		}

		err := errors.NewUnknownf("I am a teapot")
		actual := mapErrorToHTTPCode(ctx, err)
		assert.Equal(t, http.StatusTeapot, actual)
	})

	t.Run("Custom ErrorCodeMapper returning 0 falls back to default", func(t *testing.T) {
		defer func() { ErrorCodeMapper = nil }() // Reset
		ErrorCodeMapper = func(ctx context.Context, err error) int {
			return 0
		}

		err := errors.NewNotFoundf("user not found")
		actual := mapErrorToHTTPCode(ctx, err)
		assert.Equal(t, http.StatusNotFound, actual)
	})
}

func TestErrorHandler_ContextCanceled_Returns499(t *testing.T) {
	// Simulate the DS responseErrorHandler wrapping context.Canceled as ErrCodeUnknown.
	fwErr := errors.NewUnknownf("response error: %w", fmt.Errorf("wrapped: %w", context.Canceled))

	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)

	ErrorHandler(w, r, fwErr)

	require.Equal(t, StatusClientClosed, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, errors.ErrCodeCanceled, body["kind"])
}

func TestErrorHandler_ContextCanceled_AlreadyCanceled_NotDoubleWrapped(t *testing.T) {
	fwErr := errors.NewCanceledf("already canceled: %w", context.Canceled)

	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)

	ErrorHandler(w, r, fwErr)

	require.Equal(t, StatusClientClosed, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, errors.ErrCodeCanceled, body["kind"])
}

func TestErrorHandler_RegularError_Returns500(t *testing.T) {
	fwErr := errors.NewUnknownf("something went wrong")

	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)

	ErrorHandler(w, r, fwErr)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, errors.ErrCodeUnknown, body["kind"])
}

func TestErrorHandler_DeadlineExceeded_Returns500(t *testing.T) {
	// DeadlineExceeded is a server-side concern and must NOT be reclassified to 499.
	fwErr := errors.NewUnknownf("response error: %w", fmt.Errorf("wrapped: %w", context.DeadlineExceeded))

	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)

	ErrorHandler(w, r, fwErr)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}
