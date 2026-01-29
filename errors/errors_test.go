package errors_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/errors"
)

type NilWrappedError struct {
}

func (_ NilWrappedError) Error() string {
	return "nil wrapped error"
}

func (_ NilWrappedError) Unwrap() error {
	return nil
}

func TestError(t *testing.T) {
	err1 := NilWrappedError{}
	err2 := errors.Newf(errors.ErrCodeUnknown, "wrapping err1: %w", err1)
	err3 := fmt.Errorf("wrapping err2: %w", err2)
	err4 := fmt.Errorf("wrapping err3: %w", err3)
	err5 := errors.NewUnknownf("wrapping err4: %w", err4)

	require.EqualValues(
		t,
		"wrapping err4: wrapping err3: wrapping err2: {UNKNOWN} wrapping err1: nil wrapped error",
		err5.Message,
	)
	require.NotContains(t, err5.Message, "wrapped stacktrace:")
	require.True(
		t,
		strings.HasPrefix(
			err5.Error(),
			"{UNKNOWN} wrapping err4: wrapping err3: wrapping err2: {UNKNOWN} wrapping err1: nil wrapped error",
		),
		err5.Error(),
	)
	require.NotContains(t, err5.Error(), "wrapping stacktrace:")

	require.GreaterOrEqual(t, strings.Count(err5.Stacktrace(), "\n"), 3)
}

func TestErrorIs(t *testing.T) {
	tagErrStruct := &errors.Error{}

	require.ErrorIs(t, fmt.Errorf("this is a fmt wrapped error: %w", tagErrStruct), tagErrStruct)
	require.ErrorIs(t, errors.NewUnknownf("this is a fw wrapped error: %w", tagErrStruct), tagErrStruct)

	tagErrNewf := errors.Newf("MY_TAG_CODE", "This is tag error")
	require.ErrorIs(t, fmt.Errorf("this is a fmt wrapped error: %w", tagErrNewf), tagErrNewf)
	require.ErrorIs(t, errors.NewUnknownf("this is a fw wrapped error: %w", tagErrNewf), tagErrNewf)

	tagErrNewf2 := errors.Newf("MY_TAG_CODE_TWO", "This is a tag error two")
	require.ErrorIs(t, fmt.Errorf("this is a fmt wrapping multiple errors: %w, %w", tagErrNewf, tagErrNewf2), tagErrNewf)
	require.ErrorIs(t, errors.NewUnknownf("this is a fw wrapping multiple errors: %w, %w", tagErrNewf, tagErrNewf2), tagErrNewf)
	require.ErrorIs(t, errors.NewUnknownf("this is a fw wrapping multiple errors: %w, %w", tagErrNewf, tagErrNewf2), tagErrNewf2)
}

func TestErrorIsCode(t *testing.T) {
	tagErrStruct := &errors.Error{Code: "MY_TAG_CODE"}
	tagCode := tagErrStruct.Code

	require.True(t, errors.IsCode(
		fmt.Errorf("this is a fmt wrapped error: %w", tagErrStruct),
		tagCode,
	))
	require.True(t, errors.IsCode(
		errors.NewUnknownf("this is a fw wrapped error: %w", tagErrStruct),
		tagCode,
	))

	tagErrNewf := errors.Newf(tagCode, "This is tag error")
	require.True(t, errors.IsCode(
		fmt.Errorf("this is a fmt wrapped error: %w", tagErrNewf),
		tagCode,
	))
	require.True(t, errors.IsCode(
		errors.NewUnknownf("this is a fw wrapped error: %w", tagErrNewf),
		tagCode,
	))

	tagErrNewf2 := errors.Newf("MY_TAG_CODE_TWO", "This is a tag error two")
	tagCode2 := tagErrNewf2.Code
	require.True(t, errors.IsCode(
		fmt.Errorf("this is a fmt wrapping multiple errors: %w, %w", tagErrNewf, tagErrNewf2),
		tagCode2,
	))
	require.True(t, errors.IsCode(
		errors.NewUnknownf("this is a fw wrapping multiple errors: %w, %w", tagErrNewf, tagErrNewf2),
		tagCode2,
	))
	require.True(t, errors.IsCode(
		errors.NewUnknownf("this is a fw wrapping multiple errors: %w, %w", tagErrNewf, tagErrNewf2),
		tagCode2,
	))
}

func TestErrorAs(t *testing.T) {
	tagErrStruct := &errors.Error{}
	var fwErr *errors.Error

	require.ErrorAs(t, fmt.Errorf("this is a fmt wrapped error: %w", tagErrStruct), &fwErr)
	require.ErrorAs(t, errors.NewUnknownf("this is a fw wrapped error: %w", tagErrStruct), &fwErr)

	tagErrNewf := errors.Newf("MY_TAG_CODE", "This is tag error")
	require.ErrorAs(t, fmt.Errorf("this is a fmt wrapped error: %w", tagErrNewf), &fwErr)
	require.ErrorAs(t, errors.NewUnknownf("this is a fw wrapped error: %w", tagErrNewf), &fwErr)

	tagErrNewf2 := errors.Newf("MY_TAG_CODE_TWO", "This is a tag error two")
	require.ErrorAs(t, fmt.Errorf("this is a fmt wrapping multiple errors: %w, %w", tagErrNewf, tagErrNewf2), &fwErr)
	require.ErrorAs(t, errors.NewUnknownf("this is a fw wrapping multiple errors: %w, %w", tagErrNewf, tagErrNewf2), &fwErr)
	require.ErrorAs(t, errors.NewUnknownf("this is a fw wrapping multiple errors: %w, %w", tagErrNewf, tagErrNewf2), &fwErr)
}

func TestErrorAsCode(t *testing.T) {
	tagErrStruct := &errors.Error{Code: "MY_TAG_CODE"}
	tagCode := tagErrStruct.Code
	var fwErr *errors.Error

	require.True(t, errors.AsCode(fmt.Errorf("this is a fmt wrapped error: %w", tagErrStruct), &fwErr, tagCode))
	require.True(t, errors.AsCode(
		errors.NewUnknownf("this is a fw wrapped error: %w", tagErrStruct),
		&fwErr,
		tagCode,
	))

	tagErrNewf := errors.Newf(tagCode, "This is tag error")
	require.True(t, errors.AsCode(fmt.Errorf("this is a fmt wrapped error: %w", tagErrNewf), &fwErr, tagCode))
	require.True(t, errors.AsCode(
		errors.NewUnknownf("this is a fw wrapped error: %w", tagErrNewf),
		&fwErr,
		tagCode,
	))

	tagErrNewf2 := errors.Newf("MY_TAG_CODE_TWO", "This is a tag error two")
	require.True(t, errors.AsCode(
		fmt.Errorf("this is a fmt wrapping multiple errors: %w, %w", tagErrNewf, tagErrNewf2),
		&fwErr,
		tagCode,
	))
	require.True(t, errors.AsCode(
		errors.NewUnknownf("this is a fw wrapping multiple errors: %w, %w", tagErrNewf, tagErrNewf2),
		&fwErr,
		tagCode,
	))
	require.True(t, errors.AsCode(
		errors.NewUnknownf("this is a fw wrapping multiple errors: %w, %w", tagErrNewf, tagErrNewf2),
		&fwErr,
		tagCode,
	))
}

func TestWrap(t *testing.T) {
	e1 := errors.Newf(errors.ErrCodeBadState, "ate bad food")
	e2 := errors.NewUnknownf("could not eat: %w", e1)
	require.EqualValues(t, errors.ErrCodeUnknown, e2.Code)
	require.EqualValues(t, "could not eat: {BAD_STATE} ate bad food", e2.Message)
	require.EqualValues(t, e1, e2.Unwrap()[0])
	require.ErrorContains(t, e2, "{UNKNOWN} could not eat: {BAD_STATE} ate bad food")
	require.EqualValues(t, 2, strings.Count(e2.Error(), "stacktrace:"))
	require.EqualValues(t, 1, strings.Count(e2.Error(), "wrapped stacktrace:"))

	e3 := errors.Newf("MULTI_WRAP", "ate a lot of bad food: %w, %w", e1, e2)
	require.EqualValues(t, "MULTI_WRAP", e3.Code)
	require.EqualValues(t, "ate a lot of bad food: {BAD_STATE} ate bad food, {UNKNOWN} could not eat: {BAD_STATE} ate bad food", e3.Message)
	wrappedErrs := e3.Unwrap()
	require.Len(t, wrappedErrs, 2)
	require.EqualValues(t, e1, wrappedErrs[0])
	require.EqualValues(t, e2, wrappedErrs[1])
	require.Contains(t, e1.Error(), "ate bad food")
	require.Contains(t, e1.Error(), "\nstacktrace:")

	ee2 := fmt.Errorf("could not eat: %w", e1)
	ee3 := errors.NewUnknownf("my tommy aches: %w", ee2)
	require.EqualValues(t, errors.ErrCodeUnknown, ee3.Code)
	require.EqualValues(t, "my tommy aches: could not eat: {BAD_STATE} ate bad food", ee3.Message)
	require.EqualValues(t, ee2, ee3.Unwrap()[0])
}

func TestWrapParallel(t *testing.T) {
	e1 := errors.Newf(errors.ErrCodeBadState, "ate bad food")
	wg := sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e2 := errors.NewUnknownf("could not eat: %w", e1)
			require.EqualValues(t, errors.ErrCodeUnknown, e2.Code)
			require.EqualValues(t, "could not eat: {BAD_STATE} ate bad food", e2.Message)
			require.EqualValues(t, e1, e2.Unwrap()[0])

		}()
	}
	wg.Wait()
}

// Test edge case: nil error handling
func TestNilError(t *testing.T) {
	// Test that nil causes return nil from Cause
	require.Nil(t, errors.Cause(nil))
	require.Nil(t, errors.DeepCause(nil))
	require.Nil(t, errors.UnwrapMulti(nil))

	// Test that error with nil wrapped error still works
	nilWrapped := NilWrappedError{}
	require.EqualValues(t, "nil wrapped error", nilWrapped.Error())
	require.Nil(t, nilWrapped.Unwrap())

	// Test wrapping nil
	e1 := errors.Newf(errors.ErrCodeUnknown, "wrapping nothing")
	require.Empty(t, e1.Unwrap())
	require.Nil(t, e1.DeepCause())
	require.Nil(t, e1.FWCause())
	require.Nil(t, e1.FWDeepCause())
}

// Test edge case: empty error chain
func TestEmptyErrorChain(t *testing.T) {
	e := &errors.Error{
		Code:    "TEST_CODE",
		Message: "test message",
	}

	require.Empty(t, e.Unwrap())
	require.Nil(t, e.DeepCause())
	require.Nil(t, e.FWCause())
	require.Nil(t, e.FWDeepCause())
}

// Test NewBadArgumentf and other convenience constructors
func TestConvenienceConstructors(t *testing.T) {
	badArg := errors.NewBadArgumentf("invalid input: %s", "test")
	require.EqualValues(t, errors.ErrCodeBadArgument, badArg.Code)
	require.Contains(t, badArg.Message, "invalid input: test")

	validation := errors.NewValidationFailedf("field %s is required", "email")
	require.EqualValues(t, errors.ErrCodeValidationFailed, validation.Code)
	require.Contains(t, validation.Message, "field email is required")

	conflict := errors.NewConflictf("resource already exists")
	require.EqualValues(t, errors.ErrCodeConflict, conflict.Code)
	require.Contains(t, conflict.Message, "resource already exists")

	notFound := errors.NewNotFoundf("user %d not found", 123)
	require.EqualValues(t, errors.ErrCodeNotFound, notFound.Code)
	require.Contains(t, notFound.Message, "user 123 not found")

	notAuth := errors.NewNotAuthenticatedf("token expired")
	require.EqualValues(t, errors.ErrCodeNotAuthenticated, notAuth.Code)
	require.Contains(t, notAuth.Message, "token expired")

	notAllowed := errors.NewNotAllowedf("insufficient permissions")
	require.EqualValues(t, errors.ErrCodeNotAllowed, notAllowed.Code)
	require.Contains(t, notAllowed.Message, "insufficient permissions")
}

// Test IsCode with edge cases
func TestIsCodeEdgeCases(t *testing.T) {
	// Test with nil error
	require.False(t, errors.IsCode(nil, "ANY_CODE"))

	// Test with non-framework error
	stdErr := fmt.Errorf("standard error")
	require.False(t, errors.IsCode(stdErr, errors.ErrCodeUnknown))

	// Test with deeply nested framework errors
	e1 := errors.Newf("CODE1", "error 1")
	e2 := errors.Newf("CODE2", "wrapping e1: %w", e1)
	e3 := fmt.Errorf("wrapping e2: %w", e2)
	e4 := errors.Newf("CODE3", "wrapping e3: %w", e3)

	require.True(t, errors.IsCode(e4, "CODE1"))
	require.True(t, errors.IsCode(e4, "CODE2"))
	require.True(t, errors.IsCode(e4, "CODE3"))
	require.False(t, errors.IsCode(e4, "CODE4"))
}

// Test AsCode with edge cases
func TestAsCodeEdgeCases(t *testing.T) {
	var fwErr *errors.Error

	// Test with nil error
	require.False(t, errors.AsCode(nil, &fwErr, "ANY_CODE"))

	// Test with non-framework error
	stdErr := fmt.Errorf("standard error")
	require.False(t, errors.AsCode(stdErr, &fwErr, errors.ErrCodeUnknown))

	// Test with deeply nested framework errors
	e1 := errors.Newf("CODE1", "error 1")
	e2 := errors.Newf("CODE2", "wrapping e1: %w", e1)
	e3 := fmt.Errorf("wrapping e2: %w", e2)
	e4 := errors.Newf("CODE3", "wrapping e3: %w", e3)

	require.True(t, errors.AsCode(e4, &fwErr, "CODE1"))
	require.NotNil(t, fwErr)
	require.EqualValues(t, "CODE1", fwErr.Code)

	require.True(t, errors.AsCode(e4, &fwErr, "CODE2"))
	require.NotNil(t, fwErr)
	require.EqualValues(t, "CODE2", fwErr.Code)

	require.False(t, errors.AsCode(e4, &fwErr, "CODE_NOT_EXISTS"))
}

// Test Copy functionality
func TestCopy(t *testing.T) {
	e1 := errors.Newf(errors.ErrCodeBadState, "original error")
	e1.SetCodeKey("customCode")
	e1.SetMessageKey("customMessage")
	e1.SetStackKey("customStack")

	e2 := e1.Copy()

	require.EqualValues(t, e1.Code, e2.Code)
	require.EqualValues(t, e1.Message, e2.Message)
	require.NotSame(t, e1, e2)

	// Modify copy should not affect original
	e2.Code = "MODIFIED"
	require.NotEqualValues(t, e1.Code, e2.Code)
}

// Test SetCodeKey, SetMessageKey, SetStackKey with edge cases
func TestSetKeysEdgeCases(t *testing.T) {
	e := errors.Newf(errors.ErrCodeUnknown, "test error")

	// Test SetCodeKey with empty string should panic
	require.Panics(t, func() {
		e.SetCodeKey("")
	})

	// Test SetMessageKey with empty string
	e.SetMessageKey("")
	json, err := e.MarshalJSON()
	require.NoError(t, err)
	require.NotContains(t, string(json), "message")

	// Test SetStackKey with empty string
	e.SetStackKey("")
	json, err = e.MarshalJSON()
	require.NoError(t, err)
	require.NotContains(t, string(json), "stack")

	// Test custom keys
	e.SetCodeKey("error_type")
	e.SetMessageKey("error_message")
	e.SetStackKey("error_stack")
	json, err = e.MarshalJSON()
	require.NoError(t, err)
	require.Contains(t, string(json), "error_type")
	require.Contains(t, string(json), "error_message")
	require.Contains(t, string(json), "error_stack")
}

// Test SetDefaultKeys edge cases
func TestSetDefaultKeysEdgeCases(t *testing.T) {
	// Save originals
	originalCode := "kind"
	originalMessage := "message"
	originalStack := "stack"

	// Test SetDefaultCodeKey with empty should panic
	require.Panics(t, func() {
		errors.SetDefaultCodeKey("")
	})

	// Test changing defaults
	errors.SetDefaultCodeKey("custom_code")
	errors.SetDefaultMessageKey("custom_msg")
	errors.SetDefaultStackKey("custom_stack")

	e := errors.Newf("TEST", "test error")
	json, err := e.MarshalJSON()
	require.NoError(t, err)
	require.Contains(t, string(json), "custom_code")
	require.Contains(t, string(json), "custom_msg")
	require.Contains(t, string(json), "custom_stack")

	// Restore defaults
	errors.SetDefaultCodeKey(originalCode)
	errors.SetDefaultMessageKey(originalMessage)
	errors.SetDefaultStackKey(originalStack)
}

// Test LogValue
func TestLogValue(t *testing.T) {
	e := errors.Newf(errors.ErrCodeBadState, "test error")

	logValue := e.LogValue()
	require.NotNil(t, logValue)

	// Test with custom keys
	e.SetCodeKey("error_kind")
	e.SetMessageKey("error_msg")
	e.SetStackKey("error_trace")

	logValue = e.LogValue()
	require.NotNil(t, logValue)

	// Test with empty message and stack keys
	e.SetMessageKey("")
	e.SetStackKey("")
	logValue = e.LogValue()
	require.NotNil(t, logValue)
}

// Test MarshalJSON edge cases
func TestMarshalJSONEdgeCases(t *testing.T) {
	e := errors.Newf(errors.ErrCodeBadState, "test error with special chars: \n\t\"")

	jsonBytes, err := e.MarshalJSON()
	require.NoError(t, err)
	require.NotEmpty(t, jsonBytes)

	// Verify it's valid JSON
	var result map[string]string
	err = json.Unmarshal(jsonBytes, &result)
	require.NoError(t, err)
	require.Contains(t, result, "kind")
	require.EqualValues(t, errors.ErrCodeBadState, result["kind"])
}

// Test DeepCause with various nesting
func TestDeepCauseVariousNesting(t *testing.T) {
	stdErr := fmt.Errorf("root cause")
	e1 := fmt.Errorf("wrapping std: %w", stdErr)
	e2 := errors.Newf("CODE1", "wrapping e1: %w", e1)
	e3 := fmt.Errorf("wrapping e2: %w", e2)
	e4 := errors.Newf("CODE2", "wrapping e3: %w", e3)

	deepCause := errors.DeepCause(e4)
	require.NotNil(t, deepCause)
	require.EqualValues(t, stdErr, deepCause)

	// Test with nil at the end
	nilWrapped := NilWrappedError{}
	deepCauseNil := errors.DeepCause(nilWrapped)
	require.Nil(t, deepCauseNil)
}

// Test FWCause and FWDeepCause
func TestFWCauseAndFWDeepCause(t *testing.T) {
	// Create a chain: e1 -> stdErr -> e2 -> e3
	e1 := errors.Newf("CODE1", "first")
	stdErr := fmt.Errorf("standard: %w", e1)
	e2 := errors.Newf("CODE2", "second: %w", stdErr)
	e3 := errors.Newf("CODE3", "third: %w", e2)

	// Test FWCause
	fwCause := e3.FWCause()
	require.NotNil(t, fwCause)
	require.EqualValues(t, "CODE2", fwCause.Code)

	// Test FWDeepCause
	fwDeepCause := e3.FWDeepCause()
	require.NotNil(t, fwDeepCause)
	require.EqualValues(t, "CODE1", fwDeepCause.Code)

	// Test with no framework errors in chain
	stdErr1 := fmt.Errorf("error 1")
	stdErr2 := fmt.Errorf("error 2: %w", stdErr1)
	e4 := errors.Newf("CODE4", "wrapping std: %w", stdErr2)

	require.Nil(t, e4.FWCause())
	require.Nil(t, e4.FWDeepCause())
}

// Test Stacktrace and SelfStacktrace
func TestStacktraceEdgeCases(t *testing.T) {
	e := errors.Newf(errors.ErrCodeUnknown, "test")

	// Test Stacktrace
	stacktrace := e.Stacktrace()
	require.NotEmpty(t, stacktrace)
	require.Contains(t, stacktrace, "errors_test.go")

	// Test SelfStacktrace
	selfStacktrace := e.SelfStacktrace()
	require.NotEmpty(t, selfStacktrace)
	require.Contains(t, selfStacktrace, "errors_test.go")

	// Test with wrapped error
	e2 := errors.Newf("CODE2", "wrapping: %w", e)
	stacktrace2 := e2.Stacktrace()
	require.Contains(t, stacktrace2, "wrapped stacktrace:")
}

// Test Error() method behavior in different contexts
func TestErrorStringBehavior(t *testing.T) {
	e1 := errors.Newf(errors.ErrCodeBadState, "base error")

	// Direct call should include stacktrace
	errString := e1.Error()
	require.Contains(t, errString, "{BAD_STATE}")
	require.Contains(t, errString, "\nstacktrace:")

	// When wrapped by fmt.Errorf, should not include stacktrace
	wrapped := fmt.Errorf("wrapper: %w", e1)
	wrappedString := wrapped.Error()
	// The outer error string should contain the short form
	require.Contains(t, wrappedString, "{BAD_STATE}")

	// Test error message without wrapping
	e2 := errors.Newf("SIMPLE", "simple error")
	require.Contains(t, e2.Message, "simple error")
	require.NotContains(t, e2.Message, "stacktrace:")
}

// Test UnwrapMulti edge cases
func TestUnwrapMultiEdgeCases(t *testing.T) {
	// Test with nil
	require.Nil(t, errors.UnwrapMulti(nil))

	// Test with standard error
	stdErr := fmt.Errorf("standard error")
	require.Nil(t, errors.UnwrapMulti(stdErr))

	// Test with single wrapped error
	e1 := errors.Newf("CODE1", "error 1")
	wrapped := fmt.Errorf("wrapping: %w", e1)
	unwrapped := errors.UnwrapMulti(wrapped)
	require.Len(t, unwrapped, 1)
	require.EqualValues(t, e1, unwrapped[0])

	// Test with multiple wrapped errors
	e2 := errors.Newf("CODE2", "error 2")
	multiWrapped := errors.Newf("MULTI", "wrapping multiple: %w, %w", e1, e2)
	multiUnwrapped := errors.UnwrapMulti(multiWrapped)
	require.Len(t, multiUnwrapped, 2)
	require.EqualValues(t, e1, multiUnwrapped[0])
	require.EqualValues(t, e2, multiUnwrapped[1])
}

// Test Cause edge cases
func TestCauseEdgeCases(t *testing.T) {
	// Test with nil
	require.Nil(t, errors.Cause(nil))

	// Test with standard error
	stdErr := fmt.Errorf("standard error")
	require.Nil(t, errors.Cause(stdErr))

	// Test with wrapped error
	e1 := errors.Newf("CODE1", "error 1")
	wrapped := fmt.Errorf("wrapping: %w", e1)
	cause := errors.Cause(wrapped)
	require.NotNil(t, cause)
	require.EqualValues(t, e1, cause)

	// Test with multiple wrapped errors - should return first
	e2 := errors.Newf("CODE2", "error 2")
	multiWrapped := errors.Newf("MULTI", "wrapping: %w, %w", e1, e2)
	multiCause := errors.Cause(multiWrapped)
	require.NotNil(t, multiCause)
	require.EqualValues(t, e1, multiCause)
}

// Test concurrent access to error fields
func TestConcurrentErrorAccess(t *testing.T) {
	e := errors.Newf(errors.ErrCodeBadState, "concurrent test")

	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = e.Error()
			_ = e.Stacktrace()
			_ = e.SelfStacktrace()
			_ = e.LogValue()
			_, _ = e.MarshalJSON()
		}()
	}
	wg.Wait()
}

// Test wrapping with format specifiers
func TestWrappingWithFormatSpecifiers(t *testing.T) {
	e1 := errors.Newf("BASE", "base error")

	// Test with %w
	e2 := errors.Newf("WRAP", "wrapped: %w", e1)
	require.Contains(t, e2.Message, "wrapped: {BASE} base error")
	require.Len(t, e2.Unwrap(), 1)

	// Test with multiple %w
	e3 := errors.Newf("CODE3", "error 3")
	e4 := errors.Newf("MULTI_WRAP", "wrapped multiple: %w and %w", e1, e3)
	require.Contains(t, e4.Message, "wrapped multiple:")
	require.Len(t, e4.Unwrap(), 2)

	// Test with %v (should not wrap)
	e5 := errors.Newf("NO_WRAP", "using v: %v", e1)
	require.Contains(t, e5.Message, "using v:")
	require.Empty(t, e5.Unwrap())
}

// Test error codes constants
func TestErrorCodeConstants(t *testing.T) {
	require.EqualValues(t, "PANIC", errors.ErrCodePanic)
	require.EqualValues(t, "UNKNOWN", errors.ErrCodeUnknown)
	require.EqualValues(t, "BAD_ARGUMENT", errors.ErrCodeBadArgument)
	require.EqualValues(t, "BAD_STATE", errors.ErrCodeBadState)
	require.EqualValues(t, "NOT_FOUND", errors.ErrCodeNotFound)
	require.EqualValues(t, "NOT_AUTHENTICATED", errors.ErrCodeNotAuthenticated)
	require.EqualValues(t, "NOT_ALLOWED", errors.ErrCodeNotAllowed)
	require.EqualValues(t, "NOT_VALID", errors.ErrCodeValidationFailed)
	require.EqualValues(t, "CONFLICT", errors.ErrCodeConflict)
}

// Test stdlib error functions are properly exported
func TestStdlibErrorFunctions(t *testing.T) {
	e1 := errors.Newf("CODE1", "error 1")
	e2 := errors.Newf("CODE2", "error 2")

	// Test Join
	joined := errors.Join(e1, e2)
	require.NotNil(t, joined)

	// Test Is
	require.True(t, errors.Is(e1, e1))
	require.False(t, errors.Is(e1, e2))

	// Test As
	var fwErr *errors.Error
	require.True(t, errors.As(e1, &fwErr))
	require.NotNil(t, fwErr)

	// Test Unwrap
	wrapped := fmt.Errorf("wrapping: %w", e1)
	unwrapped := errors.Unwrap(wrapped)
	require.EqualValues(t, e1, unwrapped)

	// Test ErrUnsupported
	require.NotNil(t, errors.ErrUnsupported)
}

// Test multiple consecutive non-framework errors in chain
func TestMultipleConsecutiveStandardErrors(t *testing.T) {
	// Create a chain with multiple consecutive standard errors
	// e1 (fw) -> stdErr1 -> stdErr2 -> stdErr3 -> e2 (fw)
	e1 := errors.Newf("BASE_CODE", "base framework error")
	stdErr1 := fmt.Errorf("first standard error: %w", e1)
	stdErr2 := fmt.Errorf("second standard error: %w", stdErr1)
	stdErr3 := fmt.Errorf("third standard error: %w", stdErr2)
	e2 := errors.Newf("WRAPPER_CODE", "wrapping multiple standard errors: %w", stdErr3)

	// Test that FWCause can traverse through multiple standard errors
	fwCause := e2.FWCause()
	require.NotNil(t, fwCause)
	require.EqualValues(t, "BASE_CODE", fwCause.Code)

	// Test that FWDeepCause works correctly
	fwDeepCause := e2.FWDeepCause()
	require.NotNil(t, fwDeepCause)
	require.EqualValues(t, "BASE_CODE", fwDeepCause.Code)

	// Test IsCode can find the code through multiple standard errors
	require.True(t, errors.IsCode(e2, "BASE_CODE"))
	require.True(t, errors.IsCode(e2, "WRAPPER_CODE"))
	require.False(t, errors.IsCode(e2, "NONEXISTENT"))

	// Test AsCode can find and retrieve the error through multiple standard errors
	var fwErr *errors.Error
	require.True(t, errors.AsCode(e2, &fwErr, "BASE_CODE"))
	require.NotNil(t, fwErr)
	require.EqualValues(t, "BASE_CODE", fwErr.Code)

	// Test DeepCause traverses all the way down
	deepCause := errors.DeepCause(e2)
	require.NotNil(t, deepCause)
	require.EqualValues(t, e1, deepCause)

	// Create a more complex chain with multiple fw and standard errors interspersed
	// e3 (fw) -> stdErr4 -> stdErr5 -> e4 (fw) -> stdErr6 -> stdErr7 -> e5 (fw)
	e3 := errors.Newf("CODE3", "error three")
	stdErr4 := fmt.Errorf("std error 4: %w", e3)
	stdErr5 := fmt.Errorf("std error 5: %w", stdErr4)
	e4 := errors.Newf("CODE4", "error four: %w", stdErr5)
	stdErr6 := fmt.Errorf("std error 6: %w", e4)
	stdErr7 := fmt.Errorf("std error 7: %w", stdErr6)
	e5 := errors.Newf("CODE5", "error five: %w", stdErr7)

	// Test FWCause on complex chain
	fwCause5 := e5.FWCause()
	require.NotNil(t, fwCause5)
	require.EqualValues(t, "CODE4", fwCause5.Code)

	// Test FWDeepCause on complex chain
	fwDeepCause5 := e5.FWDeepCause()
	require.NotNil(t, fwDeepCause5)
	require.EqualValues(t, "CODE3", fwDeepCause5.Code)

	// Test IsCode finds all codes in complex chain
	require.True(t, errors.IsCode(e5, "CODE3"))
	require.True(t, errors.IsCode(e5, "CODE4"))
	require.True(t, errors.IsCode(e5, "CODE5"))

	// Test with only standard errors in chain (no fw errors wrapped)
	stdOnly1 := fmt.Errorf("standard only 1")
	stdOnly2 := fmt.Errorf("standard only 2: %w", stdOnly1)
	stdOnly3 := fmt.Errorf("standard only 3: %w", stdOnly2)
	e6 := errors.Newf("TOP_CODE", "top level: %w", stdOnly3)

	require.Nil(t, e6.FWCause())
	require.Nil(t, e6.FWDeepCause())
	require.True(t, errors.IsCode(e6, "TOP_CODE"))
	require.False(t, errors.IsCode(e6, "OTHER_CODE"))

	// Verify message is built correctly with multiple standard errors
	require.Contains(t, e2.Message, "wrapping multiple standard errors")
	require.Contains(t, e2.Message, "third standard error")
}
