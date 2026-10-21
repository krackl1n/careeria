package preset

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/krackl1n/careeria/ctxmeta"
	clientmw "github.com/krackl1n/careeria/http/client/middleware"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBase(t *testing.T) {
	t.Parallel()

	t.Run("panics on empty service name", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "http client preset base: empty service name", func() {
			Base(BaseConfig{})
		})
	})

	t.Run("builds and applies base middleware", func(t *testing.T) {
		t.Parallel()

		middlewares := Base(BaseConfig{
			ServiceName:       "toolbox-client",
			MetricsRegisterer: prometheus.NewRegistry(),
		})

		assert.Len(t, middlewares, 5)

		var seen *http.Request
		transport := applyMiddlewares(middlewares, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			seen = r
			return response(r, http.StatusOK), nil
		}))

		request := newRequest(http.MethodGet, "https://example.test/users").WithContext(
			ctxmeta.WithRequestID(context.Background(), "req-123"),
		)
		_, err := transport.RoundTrip(request)

		require.NoError(t, err)
		require.NotNil(t, seen)
		assert.Equal(t, "toolbox-client", seen.Header.Get("User-Agent"))
		assert.Equal(t, "req-123", seen.Header.Get(clientmw.RequestIDHeader))
	})
}

func TestExternal(t *testing.T) {
	t.Parallel()

	middlewares := External(ExternalConfig{
		BaseConfig: BaseConfig{
			ServiceName:       "external-client",
			MetricsRegisterer: prometheus.NewRegistry(),
		},
		TokenProvider: tokenProviderFunc(func(context.Context) (string, error) {
			return "token-123", nil
		}),
		APIKeyHeader: "X-API-Key",
		APIKey: func(context.Context) (string, error) {
			return "key-123", nil
		},
	})

	assert.Len(t, middlewares, 7)

	var seen *http.Request
	transport := applyMiddlewares(middlewares, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		seen = r
		return response(r, http.StatusOK), nil
	}))

	_, err := transport.RoundTrip(newRequest(http.MethodGet, "https://example.test/users"))

	require.NoError(t, err)
	require.NotNil(t, seen)
	assert.Equal(t, "Bearer token-123", seen.Header.Get("Authorization"))
	assert.Equal(t, "key-123", seen.Header.Get("X-API-Key"))
}

func TestInternal(t *testing.T) {
	t.Parallel()

	middlewares := Internal(InternalConfig{
		BaseConfig: BaseConfig{
			ServiceName:       "internal-client",
			MetricsRegisterer: prometheus.NewRegistry(),
		},
		TokenProvider: tokenProviderFunc(func(context.Context) (string, error) {
			return "token-123", nil
		}),
	})

	assert.Len(t, middlewares, 6)

	var seen *http.Request
	transport := applyMiddlewares(middlewares, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		seen = r
		return response(r, http.StatusOK), nil
	}))

	_, err := transport.RoundTrip(newRequest(http.MethodGet, "https://example.test/users"))

	require.NoError(t, err)
	require.NotNil(t, seen)
	assert.Equal(t, "Bearer token-123", seen.Header.Get("Authorization"))
}

type tokenProviderFunc func(context.Context) (string, error)

func (fn tokenProviderFunc) Token(ctx context.Context) (string, error) {
	return fn(ctx)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func applyMiddlewares(
	middlewares []func(http.RoundTripper) http.RoundTripper,
	next http.RoundTripper,
) http.RoundTripper {
	for i := len(middlewares) - 1; i >= 0; i-- {
		next = middlewares[i](next)
	}

	return next
}

func newRequest(method string, rawURL string) *http.Request {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}

	return &http.Request{
		Method: method,
		URL:    parsed,
		Header: make(http.Header),
	}
}

func response(r *http.Request, statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       http.NoBody,
		Request:    r,
	}
}
