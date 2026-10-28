package httperror_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/krackl1n/careeria/http/httperror"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorHelpers(t *testing.T) {
	t.Parallel()

	t.Run("new sets status code and message", func(t *testing.T) {
		t.Parallel()

		err := httperror.New(
			http.StatusBadRequest,
			"bad_request",
			"Bad Request",
		)

		assert.Equal(t, http.StatusBadRequest, err.Status)
		assert.Equal(t, "bad_request", err.Code)
		assert.Equal(t, "Bad Request", err.Message)
	})

	t.Run("wrap sets fields and cause", func(t *testing.T) {
		t.Parallel()

		cause := errors.New("boom")
		err := httperror.Wrap(http.StatusForbidden, "forbidden", "Forbidden", cause)

		assert.Equal(t, http.StatusForbidden, err.Status)
		assert.Equal(t, "forbidden", err.Code)
		assert.Equal(t, "Forbidden", err.Message)
		assert.Equal(t, cause, err.Err)
	})

	t.Run("with details sets details", func(t *testing.T) {
		t.Parallel()

		err := httperror.WithDetails(
			httperror.New(
				http.StatusBadRequest,
				"bad_request",
				"Bad Request",
			),
			map[string]string{"field": "email"},
		)

		assert.Equal(
			t,
			map[string]string{"field": "email"},
			err.Details,
		)
	})

	t.Run("error returns cause string when cause exists", func(t *testing.T) {
		t.Parallel()

		err := httperror.Wrap(
			http.StatusForbidden,
			"forbidden",
			"Forbidden",
			errors.New("boom"),
		)

		assert.Equal(t, "boom", err.Error())
	})

	t.Run("error returns code when cause is missing", func(t *testing.T) {
		t.Parallel()

		err := httperror.New(
			http.StatusForbidden,
			"forbidden",
			"Forbidden",
		)

		assert.Equal(t, "forbidden", err.Error())
	})

	t.Run("error returns status text when only status is set", func(t *testing.T) {
		t.Parallel()

		err := httperror.Error{
			Status: http.StatusNotFound,
		}

		assert.Equal(t, "Not Found", err.Error())
	})

	t.Run("unwrap returns cause", func(t *testing.T) {
		t.Parallel()

		cause := errors.New("boom")
		err := httperror.Wrap(
			http.StatusForbidden,
			"forbidden",
			"Forbidden",
			cause,
		)

		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("errors is works through wrapped cause", func(t *testing.T) {
		t.Parallel()

		cause := errors.New("boom")
		err := httperror.Wrap(
			http.StatusForbidden,
			"forbidden",
			"Forbidden",
			cause,
		)

		require.ErrorIs(t, err, cause)
	})
}
