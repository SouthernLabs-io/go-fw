package errors_test

import (
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
