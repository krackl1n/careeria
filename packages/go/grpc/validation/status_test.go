package validation

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testFieldError struct {
	field  string
	reason string
}

func (e testFieldError) Error() string {
	return e.reason
}

func (e testFieldError) Field() string {
	return e.field
}

func (e testFieldError) Reason() string {
	return e.reason
}

type testMultiError struct {
	errors []error
}

func (e testMultiError) Error() string {
	return "multiple validation errors"
}

func (e testMultiError) AllErrors() []error {
	return e.errors
}

func TestStatusError(t *testing.T) {
	t.Parallel()

	t.Run("adds field violations", func(t *testing.T) {
		t.Parallel()

		err := StatusError(testMultiError{errors: []error{
			testFieldError{field: "title", reason: "must be at least 3 characters"},
			testFieldError{field: "description", reason: "must not exceed 5000 characters"},
		}})

		require.Equal(t, codes.InvalidArgument, status.Code(err))
		require.Equal(t, requestValidationFailed, status.Convert(err).Message())
		details := status.Convert(err).Details()
		require.Len(t, details, 1)

		badRequest, ok := details[0].(*errdetails.BadRequest)
		require.True(t, ok)
		violations := badRequest.GetFieldViolations()
		require.Len(t, violations, 2)
		require.Equal(t, "title", violations[0].GetField())
		require.Equal(t, invalidFieldCode, violations[0].GetReason())
		require.Equal(t, "must be at least 3 characters", violations[0].GetDescription())
		require.Equal(t, "description", violations[1].GetField())
		require.Equal(t, invalidFieldCode, violations[1].GetReason())
		require.Equal(t, "must not exceed 5000 characters", violations[1].GetDescription())
	})

	t.Run("returns generic status for an unrecognized error", func(t *testing.T) {
		t.Parallel()

		err := StatusError(errors.New("invalid"))

		require.Equal(t, codes.InvalidArgument, status.Code(err))
		require.Equal(t, requestValidationFailed, status.Convert(err).Message())
		require.Empty(t, status.Convert(err).Details())
	})
}
