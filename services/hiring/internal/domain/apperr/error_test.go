package apperr

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppError(t *testing.T) {
	cause := errors.New("underlying db error")
	err := NotFound("application_not_found", "job application does not exist", cause)

	require.Equal(t, TypeNotFound, err.Type())
	require.Equal(t, "application_not_found", err.Code())
	require.Equal(t, "job application does not exist", err.Message())
	require.ErrorIs(t, err, cause)
	require.Contains(t, err.Error(), "application_not_found: job application does not exist: underlying db error")

	target := NotFound("application_not_found", "", nil)
	require.True(t, err.Is(target))

	differentCode := NotFound("different_code", "", nil)
	require.False(t, err.Is(differentCode))
}

