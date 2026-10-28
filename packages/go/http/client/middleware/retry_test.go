package middleware

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetry(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil retry function", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "http client retry middleware: nil retry function", func() {
			Retry(nil)
		})
	})

	t.Run("delegates round trip through retry function", func(t *testing.T) {
		t.Parallel()

		var called bool
		var seenMethod string

		middleware := Retry(func(
			ctx context.Context,
			req *http.Request,
			attempt func(context.Context, *http.Request) (*http.Response, error),
		) (*http.Response, error) {
			called = true
			seenMethod = req.Method

			resp, err := attempt(ctx, req)
			require.NoError(t, err)

			return resp, nil
		})

		response, err := middleware(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusAccepted,
				Body:       http.NoBody,
				Request:    r,
			}, nil
		})).RoundTrip((&http.Request{
			Method: http.MethodPost,
			Header: make(http.Header),
		}).WithContext(context.Background()))

		require.NoError(t, err)
		require.True(t, called)
		assert.Equal(t, http.MethodPost, seenMethod)
		assert.Equal(t, http.StatusAccepted, response.StatusCode)
	})

	t.Run("attempt clones request when context changes", func(t *testing.T) {
		t.Parallel()

		originalCtx := context.Background()
		attemptCtx := context.WithValue(context.Background(), struct{}{}, "value")

		req := (&http.Request{
			Method: http.MethodGet,
			Header: make(http.Header),
		}).WithContext(originalCtx)

		var seenContext context.Context

		_, err := Retry(func(
			ctx context.Context,
			req *http.Request,
			attempt func(context.Context, *http.Request) (*http.Response, error),
		) (*http.Response, error) {
			return attempt(attemptCtx, req)
		})(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			seenContext = r.Context()

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
				Request:    r,
			}, nil
		})).RoundTrip(req)

		require.NoError(t, err)
		assert.NotEqual(t, originalCtx, seenContext)
		assert.Equal(t, "value", seenContext.Value(struct{}{}))
	})
}
