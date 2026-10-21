package handler

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/krackl1n/careeria/http/httperror"
)

func TestSanitize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    ResolvedError
		expected ResolvedError
	}{
		{
			name: "invalid zero status",
			input: ResolvedError{
				Status: 0,
			},
			expected: internalError(),
		},
		{
			name: "status below client error range",
			input: ResolvedError{
				Status: 399,
			},
			expected: internalError(),
		},
		{
			name: "status above server error range",
			input: ResolvedError{
				Status: 600,
			},
			expected: internalError(),
		},
		{
			name: "bad request fills empty code and message",
			input: ResolvedError{
				Status: http.StatusBadRequest,
			},
			expected: ResolvedError{
				Status:  http.StatusBadRequest,
				Code:    "bad_request",
				Message: "Bad Request",
			},
		},
		{
			name: "unauthorized fills empty code and message",
			input: ResolvedError{
				Status: http.StatusUnauthorized,
			},
			expected: ResolvedError{
				Status:  http.StatusUnauthorized,
				Code:    "unauthorized",
				Message: "Unauthorized",
			},
		},
		{
			name: "forbidden fills empty code and message",
			input: ResolvedError{
				Status: http.StatusForbidden,
			},
			expected: ResolvedError{
				Status:  http.StatusForbidden,
				Code:    "forbidden",
				Message: "Forbidden",
			},
		},
		{
			name: "not found fills empty code and message",
			input: ResolvedError{
				Status: http.StatusNotFound,
			},
			expected: ResolvedError{
				Status:  http.StatusNotFound,
				Code:    "not_found",
				Message: "Not Found",
			},
		},
		{
			name: "conflict fills empty code and message",
			input: ResolvedError{
				Status: http.StatusConflict,
			},
			expected: ResolvedError{
				Status:  http.StatusConflict,
				Code:    "conflict",
				Message: "Conflict",
			},
		},
		{
			name: "unprocessable entity fills empty code and message",
			input: ResolvedError{
				Status: http.StatusUnprocessableEntity,
			},
			expected: ResolvedError{
				Status:  http.StatusUnprocessableEntity,
				Code:    "unprocessable_entity",
				Message: "Unprocessable Entity",
			},
		},
		{
			name: "internal server error clears unsafe fields",
			input: ResolvedError{
				Status:  http.StatusInternalServerError,
				Code:    "database_failed",
				Message: "database failed",
				Details: map[string]string{
					"query": "select secret",
				},
			},
			expected: ResolvedError{
				Status:  http.StatusInternalServerError,
				Code:    "internal_server_error",
				Message: "Internal Server Error",
			},
		},
		{
			name: "service unavailable clears unsafe fields",
			input: ResolvedError{
				Status:  http.StatusServiceUnavailable,
				Code:    "database_unavailable",
				Message: "database unavailable",
				Details: map[string]string{
					"host": "primary-db",
				},
			},
			expected: ResolvedError{
				Status:  http.StatusServiceUnavailable,
				Code:    "service_unavailable",
				Message: "Service Unavailable",
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, sanitize(tt.input))
		})
	}
}

func TestResolveHTTPError(t *testing.T) {
	t.Parallel()

	details := map[string]string{"field": "email"}

	tests := []struct {
		name     string
		err      error
		expected ResolvedError
		wantOK   bool
	}{
		{
			name: "http error value",
			err: httperror.WithDetails(
				httperror.New(
					http.StatusConflict,
					"conflict",
					"conflict",
				),
				details,
			),
			expected: ResolvedError{
				Status:  http.StatusConflict,
				Code:    "conflict",
				Message: "conflict",
				Details: details,
			},
			wantOK: true,
		},
		{
			name: "http error pointer",
			err: &httperror.Error{
				Status:  http.StatusUnauthorized,
				Code:    "unauthorized",
				Message: "unauthorized",
				Details: details,
			},
			expected: ResolvedError{
				Status:  http.StatusUnauthorized,
				Code:    "unauthorized",
				Message: "unauthorized",
				Details: details,
			},
			wantOK: true,
		},
		{
			name: "wrapped http error value",
			err: fmt.Errorf(
				"wrap: %w",
				httperror.WithDetails(
					httperror.New(
						http.StatusForbidden,
						"forbidden",
						"forbidden",
					),
					details,
				),
			),
			expected: ResolvedError{
				Status:  http.StatusForbidden,
				Code:    "forbidden",
				Message: "forbidden",
				Details: details,
			},
			wantOK: true,
		},
		{
			name: "wrapped http error pointer",
			err: fmt.Errorf(
				"wrap: %w",
				&httperror.Error{
					Status:  http.StatusTooManyRequests,
					Code:    "rate_limited",
					Message: "rate limited",
					Details: details,
				},
			),
			expected: ResolvedError{
				Status:  http.StatusTooManyRequests,
				Code:    "rate_limited",
				Message: "rate limited",
				Details: details,
			},
			wantOK: true,
		},
		{
			name:   "unknown error",
			err:    errors.New("unknown failure"),
			wantOK: false,
		},
		{
			name:   "nil error",
			err:    nil,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolved, ok := resolveHTTPError(tt.err)

			require.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.expected, resolved)
		})
	}
}
