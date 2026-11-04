package middleware

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

type idempotencyStoreMock struct {
	mu sync.Mutex

	startResult IdempotencyStartResult
	startErr    error
	finishErr   error
	failErr     error

	startCalls  int
	finishCalls int
	failCalls   int

	lastKey         string
	lastFingerprint string
	lastRecord      IdempotencyRecord
}

type failingRegisterer struct{}

func (failingRegisterer) Register(prometheus.Collector) error {
	return errors.New("register failed")
}

func (failingRegisterer) MustRegister(...prometheus.Collector) {}

func (failingRegisterer) Unregister(prometheus.Collector) bool {
	return false
}

func (m *idempotencyStoreMock) Start(ctx context.Context, key string, fingerprint string, ttl time.Duration) (IdempotencyStartResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.startCalls++
	m.lastKey = key
	m.lastFingerprint = fingerprint

	return m.startResult, m.startErr
}

func (m *idempotencyStoreMock) Finish(ctx context.Context, key string, fingerprint string, record IdempotencyRecord, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.finishCalls++
	m.lastRecord = record

	return m.finishErr
}

func (m *idempotencyStoreMock) Fail(ctx context.Context, key string, fingerprint string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failCalls++
	return m.failErr
}

func TestMetricsMiddleware(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	handler := Metrics(MetricsConfig{
		Namespace: "app",
		Subsystem: "http",
	}, registry)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users", nil)
	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusCreated, recorder.Code)

	metrics, err := registry.Gather()
	require.NoError(t, err)

	byName := make(map[string]*dto.MetricFamily, len(metrics))
	for _, metric := range metrics {
		byName[metric.GetName()] = metric
	}

	require.Contains(t, byName, "app_http_http_requests_total")
	require.Contains(t, byName, "app_http_http_request_duration_seconds")
	require.Contains(t, byName, "app_http_http_requests_in_flight")

	counterMetrics := byName["app_http_http_requests_total"].GetMetric()
	require.Len(t, counterMetrics, 1)
	require.Equal(t, 1.0, counterMetrics[0].GetCounter().GetValue())
}

func TestIdempotencyMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("passes through when key is absent", func(t *testing.T) {
		t.Parallel()

		store := &idempotencyStoreMock{}
		called := false
		handler := Idempotency(IdempotencyConfig{Store: store}, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		}))

		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		handler.ServeHTTP(recorder, req)

		require.True(t, called)
		require.Equal(t, 0, store.startCalls)
	})

	t.Run("replays stored response", func(t *testing.T) {
		t.Parallel()

		store := &idempotencyStoreMock{
			startResult: IdempotencyStartResult{
				State: IdempotencyReplay,
				Record: IdempotencyRecord{
					Status: http.StatusAccepted,
					Header: http.Header{
						"X-Trace": []string{"123"},
					},
					Body: []byte("cached"),
				},
			},
		}

		handler := Idempotency(IdempotencyConfig{Store: store}, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler must not be called")
		}))

		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set(IdempotencyKeyHeader, "key-1")
		handler.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusAccepted, recorder.Code)
		require.Equal(t, "123", recorder.Header().Get("X-Trace"))
		require.Equal(t, "true", recorder.Header().Get("Idempotent-Replayed"))
		require.Equal(t, "cached", recorder.Body.String())
	})

	t.Run("stores successful response", func(t *testing.T) {
		t.Parallel()

		store := &idempotencyStoreMock{
			startResult: IdempotencyStartResult{State: IdempotencyStarted},
		}

		handler := Idempotency(IdempotencyConfig{Store: store}, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Trace", "abc")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("created"))
		}))

		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader("payload"))
		req.Header.Set(IdempotencyKeyHeader, "key-2")
		handler.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusCreated, recorder.Code)
		require.Equal(t, 1, store.startCalls)
		require.Equal(t, 1, store.finishCalls)
		require.Equal(t, 0, store.failCalls)
		require.Equal(t, http.StatusCreated, store.lastRecord.Status)
		require.Equal(t, "abc", store.lastRecord.Header.Get("X-Trace"))
		require.Equal(t, []byte("created"), store.lastRecord.Body)
		require.NotEmpty(t, store.lastFingerprint)
	})

	t.Run("releases lock when response should not be stored", func(t *testing.T) {
		t.Parallel()

		store := &idempotencyStoreMock{
			startResult: IdempotencyStartResult{State: IdempotencyStarted},
		}

		handler := Idempotency(IdempotencyConfig{
			Store: store,
			StoreStatus: func(status int) bool {
				return false
			},
		}, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("skip"))
		}))

		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("payload"))
		req.Header.Set(IdempotencyKeyHeader, "key-3")
		handler.ServeHTTP(recorder, req)

		require.Equal(t, 0, store.finishCalls)
		require.Equal(t, 1, store.failCalls)
	})

	t.Run("returns conflict on lock state", func(t *testing.T) {
		t.Parallel()

		store := &idempotencyStoreMock{
			startResult: IdempotencyStartResult{State: IdempotencyLocked},
		}

		handler := Idempotency(IdempotencyConfig{Store: store}, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler must not be called")
		}))

		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set(IdempotencyKeyHeader, "key-4")
		handler.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusConflict, recorder.Code)
		require.Contains(t, recorder.Body.String(), `"code":"idempotency_in_progress"`)
	})
}

func TestDefaultIdempotencyFingerprint(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader("body"))
	req.Header.Set("X-User-ID", "user-1")
	req.Header.Set(TenantIDHeader, "tenant-1")

	first, err := defaultIdempotencyFingerprint(req, []byte("body"))
	require.NoError(t, err)

	second, err := defaultIdempotencyFingerprint(req, []byte("body"))
	require.NoError(t, err)

	require.Equal(t, first, second)
	require.Contains(t, first, "POST:/orders:tenant-1:user-1:")
}

func TestIdempotencyWriter(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	writer := &idempotencyWriter{
		ResponseWriter: recorder,
		maxBodyBytes:   4,
	}

	_, err := io.WriteString(writer, "abcdef")
	require.NoError(t, err)

	require.True(t, writer.overflow)
	require.Equal(t, http.StatusOK, writer.statusCode())
	require.Equal(t, "abcdef", recorder.Body.String())
}

func TestMetricsRegistrationPanicsOnUnexpectedCollectorError(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	require.NotPanics(t, func() {
		registerGauge(registry, prometheus.NewGauge(prometheus.GaugeOpts{Name: "test_gauge"}))
	})

	require.Panics(t, func() {
		registerGauge(failingRegisterer{}, prometheus.NewGauge(prometheus.GaugeOpts{Name: "test_gauge_other"}))
	})
}

func TestIdempotencyStartErrorIsMapped(t *testing.T) {
	t.Parallel()

	store := &idempotencyStoreMock{
		startErr: errors.New("storage unavailable"),
	}

	handler := Idempotency(IdempotencyConfig{Store: store}, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not be called")
	}))

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(IdempotencyKeyHeader, "key-5")
	handler.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"code":"idempotency_conflict"`)
}
