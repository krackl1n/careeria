package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/httperror"
)

func TestErrorHandlerHandle(t *testing.T) {
	t.Parallel()

	t.Run("writes resolved client error", func(t *testing.T) {
		t.Parallel()

		resolver := func(error) (handler.ResolvedError, bool) {
			return handler.ResolvedError{
				Status:  http.StatusUnprocessableEntity,
				Code:    "validation_failed",
				Message: "request validation failed",
				Details: map[string]any{
					"fields": []map[string]string{
						{
							"field":   "email",
							"code":    "email",
							"message": "must be a valid email",
						},
					},
				},
			}, true
		}

		var logs bytes.Buffer

		errorHandler := handler.NewErrorHandler(
			slog.New(slog.NewJSONHandler(&logs, nil)),
			resolver,
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPost,
			"/users",
			nil,
		)
		request.Header.Set(
			handler.RequestIDHeader,
			"request-123",
		)

		errorHandler.Handle(
			recorder,
			request,
			errors.New("validation failed"),
		)

		assert.Equal(
			t,
			http.StatusUnprocessableEntity,
			recorder.Code,
		)
		assert.Equal(
			t,
			"application/json; charset=utf-8",
			recorder.Header().Get("Content-Type"),
		)
		assert.Empty(t, logs.String())

		var response struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
				Details struct {
					Fields []struct {
						Field   string `json:"field"`
						Code    string `json:"code"`
						Message string `json:"message"`
					} `json:"fields"`
				} `json:"details"`
			} `json:"error"`
			RequestID string `json:"request_id"`
		}

		require.NoError(
			t,
			json.Unmarshal(recorder.Body.Bytes(), &response),
		)

		assert.Equal(
			t,
			"validation_failed",
			response.Error.Code,
		)
		assert.Equal(
			t,
			"request validation failed",
			response.Error.Message,
		)
		assert.Equal(t, "request-123", response.RequestID)

		require.Len(t, response.Error.Details.Fields, 1)
		assert.Equal(
			t,
			"email",
			response.Error.Details.Fields[0].Field,
		)
		assert.Equal(
			t,
			"email",
			response.Error.Details.Fields[0].Code,
		)
		assert.Equal(
			t,
			"must be a valid email",
			response.Error.Details.Fields[0].Message,
		)
	})

	t.Run("uses default resolver", func(t *testing.T) {
		t.Parallel()

		errorHandler := handler.NewErrorHandler(
			slog.New(
				slog.NewTextHandler(
					&bytes.Buffer{},
					nil,
				),
			),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPost,
			"/users",
			nil,
		)

		errorHandler.Handle(
			recorder,
			request,
			&json.SyntaxError{Offset: 4},
		)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.JSONEq(
			t,
			`{
				"error": {
					"code": "invalid_json",
					"message": "request body contains invalid JSON"
				}
			}`,
			recorder.Body.String(),
		)
	})

	t.Run("uses typed http error resolver", func(t *testing.T) {
		t.Parallel()

		errorHandler := handler.NewErrorHandler(
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/users", nil)

		errorHandler.Handle(
			recorder,
			request,
			httperror.WithDetails(
				httperror.New(
					http.StatusConflict,
					"user_conflict",
					"user already exists",
				),
				map[string]string{"field": "email"},
			),
		)

		assert.Equal(t, http.StatusConflict, recorder.Code)
		assert.JSONEq(
			t,
			`{
				"error": {
					"code": "user_conflict",
					"message": "user already exists",
					"details": {
						"field": "email"
					}
				}
			}`,
			recorder.Body.String(),
		)
	})

	t.Run("uses internal fallback for unknown error", func(t *testing.T) {
		t.Parallel()

		var logs bytes.Buffer

		errorHandler := handler.NewErrorHandler(
			slog.New(slog.NewJSONHandler(&logs, nil)),
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodGet,
			"/users",
			nil,
		)

		errorHandler.Handle(
			recorder,
			request,
			errors.New("database connection failed"),
		)

		assert.Equal(
			t,
			http.StatusInternalServerError,
			recorder.Code,
		)
		assert.JSONEq(
			t,
			`{
				"error": {
					"code": "internal_server_error",
					"message": "Internal Server Error"
				}
			}`,
			recorder.Body.String(),
		)
		assert.NotContains(
			t,
			recorder.Body.String(),
			"database connection failed",
		)
		assert.Contains(
			t,
			logs.String(),
			"database connection failed",
		)
	})

	t.Run("preserves and sanitizes service unavailable", func(t *testing.T) {
		t.Parallel()

		resolver := func(error) (handler.ResolvedError, bool) {
			return handler.ResolvedError{
				Status:  http.StatusServiceUnavailable,
				Code:    "database_unavailable",
				Message: "primary database is unavailable",
				Details: map[string]string{
					"host": "internal-database",
				},
			}, true
		}

		var logs bytes.Buffer

		errorHandler := handler.NewErrorHandler(
			slog.New(slog.NewJSONHandler(&logs, nil)),
			resolver,
		)

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodGet,
			"/users",
			nil,
		)

		errorHandler.Handle(
			recorder,
			request,
			errors.New("connection refused"),
		)

		assert.Equal(
			t,
			http.StatusServiceUnavailable,
			recorder.Code,
		)
		assert.JSONEq(
			t,
			`{
				"error": {
					"code": "service_unavailable",
					"message": "Service Unavailable"
				}
			}`,
			recorder.Body.String(),
		)
		assert.NotContains(
			t,
			recorder.Body.String(),
			"internal-database",
		)
		assert.NotContains(
			t,
			recorder.Body.String(),
			"primary database",
		)
		assert.Contains(t, logs.String(), "connection refused")
	})

	t.Run("sanitizes internal server error", func(t *testing.T) {
		t.Parallel()

		resolver := func(error) (handler.ResolvedError, bool) {
			return handler.ResolvedError{
				Status:  http.StatusInternalServerError,
				Code:    "database_failed",
				Message: "database query failed",
				Details: map[string]string{
					"query": "select secret",
				},
			}, true
		}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodGet,
			"/users",
			nil,
		)

		errorHandler := handler.NewErrorHandler(
			slog.New(
				slog.NewTextHandler(
					&bytes.Buffer{},
					nil,
				),
			),
			resolver,
		)

		errorHandler.Handle(
			recorder,
			request,
			errors.New("internal failure"),
		)

		assert.Equal(
			t,
			http.StatusInternalServerError,
			recorder.Code,
		)
		assert.JSONEq(
			t,
			`{
				"error": {
					"code": "internal_server_error",
					"message": "Internal Server Error"
				}
			}`,
			recorder.Body.String(),
		)
		assert.NotContains(
			t,
			recorder.Body.String(),
			"database query failed",
		)
		assert.NotContains(
			t,
			recorder.Body.String(),
			"select secret",
		)
	})

	t.Run("replaces invalid HTTP status with internal error", func(t *testing.T) {
		t.Parallel()

		resolver := func(error) (handler.ResolvedError, bool) {
			return handler.ResolvedError{
				Status:  http.StatusOK,
				Code:    "invalid_status",
				Message: "invalid status",
			}, true
		}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodGet,
			"/resource",
			nil,
		)

		handler.NewErrorHandler(
			slog.New(
				slog.NewTextHandler(
					&bytes.Buffer{},
					nil,
				),
			),
			resolver,
		).Handle(
			recorder,
			request,
			errors.New("failure"),
		)

		assert.Equal(
			t,
			http.StatusInternalServerError,
			recorder.Code,
		)
		assert.JSONEq(
			t,
			`{
				"error": {
					"code": "internal_server_error",
					"message": "Internal Server Error"
				}
			}`,
			recorder.Body.String(),
		)
	})

	t.Run("fills empty client error fields", func(t *testing.T) {
		t.Parallel()

		resolver := func(error) (handler.ResolvedError, bool) {
			return handler.ResolvedError{
				Status: http.StatusForbidden,
			}, true
		}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodGet,
			"/private",
			nil,
		)

		handler.NewErrorHandler(
			slog.New(
				slog.NewTextHandler(
					&bytes.Buffer{},
					nil,
				),
			),
			resolver,
		).Handle(
			recorder,
			request,
			errors.New("forbidden"),
		)

		assert.Equal(t, http.StatusForbidden, recorder.Code)
		assert.JSONEq(
			t,
			`{
				"error": {
					"code": "forbidden",
					"message": "Forbidden"
				}
			}`,
			recorder.Body.String(),
		)
	})

	t.Run("does nothing for nil error", func(t *testing.T) {
		t.Parallel()

		resolverCalled := false

		resolver := func(error) (handler.ResolvedError, bool) {
			resolverCalled = true
			return handler.ResolvedError{}, false
		}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodGet,
			"/users",
			nil,
		)

		handler.NewErrorHandler(
			nil,
			resolver,
		).Handle(
			recorder,
			request,
			nil,
		)

		assert.False(t, resolverCalled)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Empty(t, recorder.Body.String())
	})
}

func TestNewErrorHandlerResolverOrder(t *testing.T) {
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

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/resource",
		nil,
	)

	handler.NewErrorHandler(
		slog.New(
			slog.NewTextHandler(
				&bytes.Buffer{},
				nil,
			),
		),
		first,
		second,
		third,
	).Handle(
		recorder,
		request,
		errors.New("failure"),
	)

	assert.Equal(t, []string{"first", "second"}, calls)
	assert.Equal(t, http.StatusConflict, recorder.Code)
}

func TestNewErrorHandlerResolverChain(t *testing.T) {
	t.Parallel()

	t.Run("custom resolver has priority over http error", func(t *testing.T) {
		t.Parallel()

		resolver := func(error) (handler.ResolvedError, bool) {
			return handler.ResolvedError{
				Status:  http.StatusTeapot,
				Code:    "custom_error",
				Message: "custom error",
			}, true
		}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/resource", nil)

		handler.NewErrorHandler(
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			resolver,
		).Handle(
			recorder,
			request,
			httperror.New(
				http.StatusConflict,
				"conflict",
				"Conflict",
			),
		)

		assert.Equal(t, http.StatusTeapot, recorder.Code)
		assert.JSONEq(
			t,
			`{
				"error": {
					"code": "custom_error",
					"message": "custom error"
				}
			}`,
			recorder.Body.String(),
		)
	})

	t.Run("http error resolves before default resolver", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/resource", nil)

		handler.NewErrorHandler(
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		).Handle(
			recorder,
			request,
			httperror.Wrap(
				http.StatusConflict,
				"explicit_conflict",
				"explicit conflict",
				context.Canceled,
			),
		)

		assert.Equal(t, http.StatusConflict, recorder.Code)
		assert.JSONEq(
			t,
			`{
				"error": {
					"code": "explicit_conflict",
					"message": "explicit conflict"
				}
			}`,
			recorder.Body.String(),
		)
	})

	t.Run("default resolver handles standard error", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/resource", nil)

		handler.NewErrorHandler(
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		).Handle(
			recorder,
			request,
			context.DeadlineExceeded,
		)

		assert.Equal(t, http.StatusGatewayTimeout, recorder.Code)
		assert.JSONEq(
			t,
			`{
				"error": {
					"code": "gateway_timeout",
					"message": "Gateway Timeout"
				}
			}`,
			recorder.Body.String(),
		)
	})

	t.Run("unknown error falls back to internal error", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/resource", nil)

		handler.NewErrorHandler(
			slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		).Handle(
			recorder,
			request,
			errors.New("unknown failure"),
		)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.JSONEq(
			t,
			`{
				"error": {
					"code": "internal_server_error",
					"message": "Internal Server Error"
				}
			}`,
			recorder.Body.String(),
		)
	})
}
