package panics_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/panics"
)

func TestDeferredPanicToError(t *testing.T) {
	f := func(id int) (err error) {
		defer panics.DeferredPanicToError(&err, "something bad happened while working: %v", id)
		switch {
		case id == 0:
			panic("id is zero")
		case id == 1:
			err = errors.NewUnknownf("id is one")
			panic("id is one")
		case id == 2:
			err = errors.NewUnknownf("id is even")
		}
		return err
	}

	require.NotPanics(t, func() {
		err := f(0)
		require.Error(t, err)
		require.True(t, errors.IsCode(err, errors.ErrCodePanic))
		require.ErrorContains(t, err, "something bad happened while working: 0")
		require.ErrorContains(t, err, "panic: id is zero")

		err = f(1)
		require.Error(t, err)
		require.True(t, errors.IsCode(err, errors.ErrCodePanic))
		require.True(t, errors.IsCode(err, errors.ErrCodeUnknown))
		require.ErrorContains(t, err, "something bad happened while working: 1")
		require.ErrorContains(t, err, "panic: id is one")
		require.ErrorContains(t, err, "hidden error: {UNKNOWN} id is one")

		err = f(2)
		require.Error(t, err)
		require.False(t, errors.IsCode(err, errors.ErrCodePanic))
		require.True(t, errors.IsCode(err, errors.ErrCodeUnknown))
		require.ErrorContains(t, err, "id is even")

		err = f(3)
		require.NoError(t, err)
	})
}
