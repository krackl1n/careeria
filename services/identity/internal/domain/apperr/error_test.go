package apperr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	"github.com/stretchr/testify/require"
)

func TestErrorCanBeClassifiedThroughWrapping(t *testing.T) {
	t.Parallel()

	cause := errors.New("duplicate key")
	domainErr := apperr.New(
		apperr.TypeConflict,
		"identity_email_taken",
		"identity email is already attached",
		cause,
	)
	err := fmt.Errorf("create identity: %w", domainErr)

	require.True(t, apperr.IsType(err, apperr.TypeConflict))
	require.True(t, apperr.IsCode(err, "identity_email_taken"))
	require.ErrorIs(t, err, cause)
	require.Equal(t, "identity_email_taken: identity email is already attached: duplicate key", domainErr.Error())
}

func TestErrorDefaultsAreSafe(t *testing.T) {
	t.Parallel()

	err := apperr.New(apperr.TypeUnknown, " ", " ", nil)
	require.Equal(t, apperr.TypeInternal, err.Type())
	require.Equal(t, "internal_error", err.Code())
	require.Equal(t, "internal error", err.Message())
}
