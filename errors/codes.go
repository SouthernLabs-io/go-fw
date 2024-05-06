package errors

// Common errors
const (
	// ErrCodePanic is used when a panic is caught, the original panic error will be wrapped in the new error
	ErrCodePanic = "PANIC"

	// ErrCodeUnknown is used when an error occurs that is not known
	ErrCodeUnknown = "UNKNOWN"

	// ErrCodeBadArgument is used when an argument is invalid. This should be used when the argument is provided by
	// an external source. This error will be mapped to HTTP 409.
	ErrCodeBadArgument = "BAD_ARGUMENT"

	// ErrCodeBadState is used when the application is in a state that is not expected, this can be used when
	// the configuration is invalid. This error will be mapped to HTTP 500.
	ErrCodeBadState = "BAD_STATE"

	// ErrCodeNotFound is used when a resource is not found. This error will be mapped to HTTP 404.
	ErrCodeNotFound = "NOT_FOUND"

	// ErrCodeNotAuthenticated is used when the caller is not authenticated. This error will be mapped to HTTP 401.
	ErrCodeNotAuthenticated = "NOT_AUTHENTICATED"

	// ErrCodeNotAllowed is used when the caller is not allowed to perform the action. This error will be mapped to HTTP 403.
	ErrCodeNotAllowed = "NOT_ALLOWED"

	// ErrCodeValidationFailed is used when a validation error occurs. This error will be mapped to HTTP 422.
	ErrCodeValidationFailed = "NOT_VALID"

	// ErrCodeConflict is used when there is a conflict with the current state. This error will be mapped to HTTP 409.
	ErrCodeConflict = "CONFLICT"
)
