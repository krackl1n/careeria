package middleware

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tokenProviderFunc func(context.Context) (string, error)

func (fn tokenProviderFunc) Token(ctx context.Context) (string, error) {
	return fn(ctx)
}

func TestBearerToken(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil provider", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "http client bearer token middleware: nil token provider", func() {
			BearerToken(nil)
		})
	})

	t.Run("sets authorization header on cloned request", func(t *testing.T) {
		t.Parallel()

		original := newRequest(http.MethodGet, "https://example.test/users")
		var seen *http.Request

		resp, err := BearerToken(tokenProviderFunc(func(context.Context) (string, error) {
			return "token-123", nil
		}))(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			seen = r
			return response(r, http.StatusNoContent), nil
		})).RoundTrip(original)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		assert.Equal(t, "Bearer token-123", seen.Header.Get("Authorization"))
		assert.Empty(t, original.Header.Get("Authorization"))
	})

	t.Run("passes through empty token", func(t *testing.T) {
		t.Parallel()

		original := newRequest(http.MethodGet, "https://example.test/users")
		var seen *http.Request

		_, err := BearerToken(tokenProviderFunc(func(context.Context) (string, error) {
			return "", nil
		}))(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			seen = r
			return response(r, http.StatusNoContent), nil
		})).RoundTrip(original)

		require.NoError(t, err)
		assert.Same(t, original, seen)
	})

	t.Run("returns provider error", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("token failed")

		_, err := BearerToken(tokenProviderFunc(func(context.Context) (string, error) {
			return "", expectedErr
		}))(roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("next must not be called")
			return nil, nil
		})).RoundTrip(newRequest(http.MethodGet, "https://example.test/users"))

		require.ErrorIs(t, err, expectedErr)
	})
}

func TestAPIKey(t *testing.T) {
	t.Parallel()

	t.Run("panics on invalid config", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "http client api key middleware: empty header", func() {
			APIKey("", func(context.Context) (string, error) { return "key", nil })
		})
		require.PanicsWithValue(t, "http client api key middleware: nil provider", func() {
			APIKey("X-API-Key", nil)
		})
	})

	t.Run("sets configured header", func(t *testing.T) {
		t.Parallel()

		original := newRequest(http.MethodGet, "https://example.test/users")
		var seen *http.Request

		_, err := APIKey("X-API-Key", func(context.Context) (string, error) {
			return "key-123", nil
		})(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			seen = r
			return response(r, http.StatusOK), nil
		})).RoundTrip(original)

		require.NoError(t, err)
		assert.Equal(t, "key-123", seen.Header.Get("X-API-Key"))
		assert.Empty(t, original.Header.Get("X-API-Key"))
	})
}

func TestRequestID(t *testing.T) {
	t.Parallel()

	t.Run("adds generated request id", func(t *testing.T) {
		t.Parallel()

		ctx := ctxmeta.WithRequestID(context.Background(), "req-123")
		ctx = ctxmeta.WithCorrelationID(ctx, "corr-123")
		original := newRequest(http.MethodGet, "https://example.test/users").WithContext(ctx)
		var seen *http.Request

		_, err := RequestID(RequestIDConfig{})(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			seen = r
			return response(r, http.StatusOK), nil
		})).RoundTrip(original)

		require.NoError(t, err)
		assert.Equal(t, "req-123", seen.Header.Get(RequestIDHeader))
		assert.Equal(t, "corr-123", seen.Header.Get(CorrelationIDHeader))
		assert.Empty(t, original.Header.Get(RequestIDHeader))
	})

	t.Run("preserves existing request id", func(t *testing.T) {
		t.Parallel()

		original := newRequest(http.MethodGet, "https://example.test/users")
		original.Header.Set(RequestIDHeader, "existing")
		var seen *http.Request

		_, err := RequestID(RequestIDConfig{})(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			seen = r
			return response(r, http.StatusOK), nil
		})).RoundTrip(original)

		require.NoError(t, err)
		assert.Same(t, original, seen)
		assert.Equal(t, "existing", seen.Header.Get(RequestIDHeader))
	})
}

func TestUserAgent(t *testing.T) {
	t.Parallel()

	t.Run("panics on empty user agent", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "http client user agent middleware: empty user agent", func() {
			UserAgent("")
		})
	})

	t.Run("sets user agent when missing", func(t *testing.T) {
		t.Parallel()

		original := newRequest(http.MethodGet, "https://example.test/users")
		var seen *http.Request

		_, err := UserAgent("toolbox-client")(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			seen = r
			return response(r, http.StatusOK), nil
		})).RoundTrip(original)

		require.NoError(t, err)
		assert.Equal(t, "toolbox-client", seen.Header.Get("User-Agent"))
		assert.Empty(t, original.Header.Get("User-Agent"))
	})
}

func TestLogging(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		err        error
		wantLevel  string
		wantMsg    string
	}{
		{
			name:       "success logs info",
			statusCode: http.StatusOK,
			wantLevel:  "INFO",
			wantMsg:    "http client request completed",
		},
		{
			name:       "server error logs error",
			statusCode: http.StatusBadGateway,
			wantLevel:  "ERROR",
			wantMsg:    "http client request completed",
		},
		{
			name:      "round trip error logs failure",
			err:       errors.New("dial failed"),
			wantLevel: "ERROR",
			wantMsg:   "http client request failed",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, nil))
			req := newRequest(http.MethodGet, "https://example.test/users?token=secret")
			req.Header.Set(RequestIDHeader, "req-123")

			_, err := Logging(logger)(roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if tt.err != nil {
					return nil, tt.err
				}

				return response(r, tt.statusCode), nil
			})).RoundTrip(req)

			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}

			assert.Contains(t, logs.String(), tt.wantLevel)
			assert.Contains(t, logs.String(), tt.wantMsg)
			assert.Contains(t, logs.String(), "https://example.test/users")
			assert.NotContains(t, logs.String(), "token=secret")
			assert.Contains(t, logs.String(), "req-123")
		})
	}
}

func TestMetrics(t *testing.T) {
	t.Parallel()

	t.Run("records requests and reuses registered collectors", func(t *testing.T) {
		t.Parallel()

		registerer := prometheus.NewRegistry()
		cfg := MetricsConfig{
			Registerer: registerer,
			Target:     "users",
		}

		mw := Metrics(cfg)
		_ = Metrics(cfg)

		_, err := mw(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return response(r, http.StatusCreated), nil
		})).RoundTrip(newRequest(http.MethodPost, "https://example.test/users"))

		require.NoError(t, err)

		metricFamilies, err := registerer.Gather()
		require.NoError(t, err)

		assertMetricValue(t, metricFamilies, "http_client_requests_total", 1)
	})

	t.Run("normalizes unknown method", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, http.MethodGet, normalizeMethod(http.MethodGet))
		assert.Equal(t, "OTHER", normalizeMethod("PURGE"))
	})
}

func TestHelpers(t *testing.T) {
	t.Parallel()

	req := newRequest(http.MethodGet, "https://example.test/users?token=secret")
	assert.Equal(t, "https://example.test/users", sanitizedURL(req))
	assert.Equal(t, "example.test", host(req))
	assert.Empty(t, sanitizedURL(nil))
	assert.Empty(t, host(nil))

	clone := cloneRequest(req)
	clone.Header.Set("X-Test", "value")
	assert.Empty(t, req.Header.Get("X-Test"))
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

func assertMetricValue(
	t *testing.T,
	families []*dto.MetricFamily,
	name string,
	expected float64,
) {
	t.Helper()

	for _, family := range families {
		if family.GetName() != name {
			continue
		}

		var total float64
		for _, metric := range family.GetMetric() {
			total += metric.GetCounter().GetValue()
		}

		assert.Equal(t, expected, total)
		return
	}

	require.Failf(t, "missing metric", "metric %q was not gathered", name)
}
