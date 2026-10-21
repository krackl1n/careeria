package handler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/krackl1n/careeria/http/handler"
)

type fakeErrorHandler struct {
	called bool
	err    error
}

func (h *fakeErrorHandler) Handle(
	_ http.ResponseWriter,
	_ *http.Request,
	err error,
) {
	h.called = true
	h.err = err
}

func TestAdapt(t *testing.T) {
	t.Parallel()

	t.Run("serves successful handler", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodGet,
			"/health",
			nil,
		)

		adapted := handler.Adapt(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) error {
				w.WriteHeader(http.StatusNoContent)
				return nil
			},
			nil,
		)

		adapted.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusNoContent, recorder.Code)
		assert.Empty(t, recorder.Body.String())
	})

	t.Run("does not call error handler when handler returns nil and writes nothing", func(t *testing.T) {
		t.Parallel()

		errorHandler := &fakeErrorHandler{}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodGet,
			"/noop",
			nil,
		)

		handler.Adapt(
			func(
				http.ResponseWriter,
				*http.Request,
			) error {
				return nil
			},
			errorHandler,
		).ServeHTTP(recorder, request)

		assert.False(t, errorHandler.called)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Empty(t, recorder.Body.String())
	})

	t.Run("passes returned error to error handler", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("handler failed")
		errorHandler := &fakeErrorHandler{}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPost,
			"/users",
			nil,
		)

		handler.Adapt(
			func(
				http.ResponseWriter,
				*http.Request,
			) error {
				return expectedErr
			},
			errorHandler,
		).ServeHTTP(recorder, request)

		require.True(t, errorHandler.called)
		require.ErrorIs(t, errorHandler.err, expectedErr)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("does not call error handler when handler writes response and returns nil", func(t *testing.T) {
		t.Parallel()

		errorHandler := &fakeErrorHandler{}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPost,
			"/jobs",
			nil,
		)

		handler.Adapt(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) error {
				w.WriteHeader(http.StatusAccepted)
				return nil
			},
			errorHandler,
		).ServeHTTP(recorder, request)

		assert.False(t, errorHandler.called)
		assert.Equal(t, http.StatusAccepted, recorder.Code)
	})

	t.Run("does not call error handler after WriteHeader", func(t *testing.T) {
		t.Parallel()

		errorHandler := &fakeErrorHandler{}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPost,
			"/jobs",
			nil,
		)

		handler.Adapt(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) error {
				w.WriteHeader(http.StatusAccepted)
				return errors.New("late failure")
			},
			errorHandler,
		).ServeHTTP(recorder, request)

		assert.False(t, errorHandler.called)
		assert.Equal(t, http.StatusAccepted, recorder.Code)
	})

	t.Run("does not call error handler after Write", func(t *testing.T) {
		t.Parallel()

		errorHandler := &fakeErrorHandler{}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPost,
			"/jobs",
			nil,
		)

		handler.Adapt(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) error {
				_, err := w.Write([]byte("accepted"))
				require.NoError(t, err)

				return errors.New("late failure")
			},
			errorHandler,
		).ServeHTTP(recorder, request)

		assert.False(t, errorHandler.called)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "accepted", recorder.Body.String())
	})

	t.Run("uses default error handler", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodGet,
			"/resource",
			nil,
		)

		handler.Adapt(
			func(
				http.ResponseWriter,
				*http.Request,
			) error {
				return errors.New("unexpected failure")
			},
			nil,
		).ServeHTTP(recorder, request)

		assert.Equal(
			t,
			http.StatusInternalServerError,
			recorder.Code,
		)
		assert.Equal(
			t,
			"application/json; charset=utf-8",
			recorder.Header().Get("Content-Type"),
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

	t.Run("panics on nil handler", func(t *testing.T) {
		t.Parallel()

		assert.PanicsWithValue(
			t,
			"http handler adapter: nil handler",
			func() {
				handler.Adapt(nil, nil)
			},
		)
	})
}
