package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/krackl1n/careeria/http/handler"
)

func TestChain(t *testing.T) {
	t.Parallel()

	t.Run("returns first successfully resolved error", func(t *testing.T) {
		t.Parallel()

		var calls []string

		first := func(error) (handler.ResolvedError, bool) {
			calls = append(calls, "first")

			return handler.ResolvedError{}, false
		}

		second := func(error) (handler.ResolvedError, bool) {
			calls = append(calls, "second")

			return handler.ResolvedError{
				Status:  http.StatusConflict,
				Code:    "conflict",
				Message: "Conflict",
			}, true
		}

		third := func(error) (handler.ResolvedError, bool) {
			calls = append(calls, "third")

			return handler.ResolvedError{
				Status:  http.StatusNotFound,
				Code:    "not_found",
				Message: "Not Found",
			}, true
		}

		resolved, ok := handler.Chain(
			first,
			nil,
			second,
			third,
		)(errors.New("failure"))

		require.True(t, ok)
		assert.Equal(t, []string{"first", "second"}, calls)
		assert.Equal(t, handler.ResolvedError{
			Status:  http.StatusConflict,
			Code:    "conflict",
			Message: "Conflict",
		}, resolved)
	})

	t.Run("returns false when no resolver matches", func(t *testing.T) {
		t.Parallel()

		resolved, ok := handler.Chain(
			nil,
			func(error) (handler.ResolvedError, bool) {
				return handler.ResolvedError{}, false
			},
		)(errors.New("failure"))

		assert.False(t, ok)
		assert.Empty(t, resolved)
	})
}

func TestDefaultResolver(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected handler.ResolvedError
	}{
		{
			name: "payload too large",
			err: &http.MaxBytesError{
				Limit: 1024,
			},
			expected: handler.ResolvedError{
				Status:  http.StatusRequestEntityTooLarge,
				Code:    "payload_too_large",
				Message: "request body is too large",
			},
		},
		{
			name: "JSON syntax error",
			err: &json.SyntaxError{
				Offset: 8,
			},
			expected: handler.ResolvedError{
				Status:  http.StatusBadRequest,
				Code:    "invalid_json",
				Message: "request body contains invalid JSON",
			},
		},
		{
			name: "JSON type error",
			err: &json.UnmarshalTypeError{
				Value:  "string",
				Type:   reflect.TypeOf(int(0)),
				Offset: 12,
				Field:  "age",
			},
			expected: handler.ResolvedError{
				Status:  http.StatusBadRequest,
				Code:    "invalid_json_type",
				Message: "request body contains an invalid value type",
			},
		},
		{
			name: "empty request body",
			err:  io.EOF,
			expected: handler.ResolvedError{
				Status:  http.StatusBadRequest,
				Code:    "empty_request_body",
				Message: "request body is required",
			},
		},
		{
			name: "deadline exceeded",
			err:  context.DeadlineExceeded,
			expected: handler.ResolvedError{
				Status: http.StatusGatewayTimeout,
				Code:   "gateway_timeout",
				Message: http.StatusText(
					http.StatusGatewayTimeout,
				),
			},
		},
		{
			name: "request canceled",
			err:  context.Canceled,
			expected: handler.ResolvedError{
				Status:  499,
				Code:    "client_closed_request",
				Message: "Client Closed Request",
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolved, ok := handler.DefaultResolver(tt.err)

			require.True(t, ok)
			assert.Equal(t, tt.expected, resolved)
		})
	}
}

func TestDefaultResolverWrappedError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name: "wrapped EOF",
			err: fmt.Errorf(
				"decode request: %w",
				io.EOF,
			),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "empty_request_body",
		},
		{
			name: "wrapped deadline",
			err: fmt.Errorf(
				"execute request: %w",
				context.DeadlineExceeded,
			),
			expectedStatus: http.StatusGatewayTimeout,
			expectedCode:   "gateway_timeout",
		},
		{
			name: "wrapped canceled context",
			err: fmt.Errorf(
				"execute request: %w",
				context.Canceled,
			),
			expectedStatus: 499,
			expectedCode:   "client_closed_request",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolved, ok := handler.DefaultResolver(tt.err)

			require.True(t, ok)
			assert.Equal(t, tt.expectedStatus, resolved.Status)
			assert.Equal(t, tt.expectedCode, resolved.Code)
		})
	}
}

func TestDefaultResolverReturnsFalse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "nil error",
			err:  nil,
		},
		{
			name: "unknown error",
			err:  errors.New("unexpected failure"),
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolved, ok := handler.DefaultResolver(tt.err)

			assert.False(t, ok)
			assert.Empty(t, resolved)
		})
	}
}
