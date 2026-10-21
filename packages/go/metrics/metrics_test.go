package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/require"
)

func TestConfigAndDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig("svc", ":9090")
	require.True(t, cfg.Enabled)
	require.Equal(t, "svc", cfg.Namespace)
	require.Equal(t, ":9090", cfg.Addr)
	require.Equal(t, 5*time.Second, cfg.ReadHeaderTimeout)

	require.EqualError(t, Config{Enabled: true}.Validate(), "addr is required")
	require.NoError(t, Config{Enabled: false}.Validate())
	require.NoError(t, cfg.Validate())
}

func TestNewMetrics(t *testing.T) {
	t.Parallel()

	t.Run("disabled", func(t *testing.T) {
		t.Parallel()
		m, err := NewMetrics(Config{Enabled: false})
		require.NoError(t, err)
		require.False(t, m.Enabled())
		require.Empty(t, m.Namespace())
	})

	t.Run("enabled", func(t *testing.T) {
		t.Parallel()
		m, err := NewMetrics(DefaultConfig("svc", ":9090"))
		require.NoError(t, err)
		require.True(t, m.Enabled())
		require.Equal(t, "svc", m.Namespace())
		require.NotNil(t, m.Registry())
	})
}

func TestMetricsRegisterAndHandler(t *testing.T) {
	t.Parallel()

	m, err := NewMetrics(DefaultConfig("svc", ":9090"))
	require.NoError(t, err)

	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "custom_counter_total",
		Help: "custom",
	})
	require.NoError(t, m.Register(counter))
	counter.Inc()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	m.Handler().ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "custom_counter_total")
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/plain")

	var nilMetrics *Metrics
	require.NoError(t, nilMetrics.Register(counter))
	require.NotPanics(t, func() {
		nilMetrics.MustRegister(counter)
	})
}

func TestServerAndComponent(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig("svc", ":9090")
	m, err := NewMetrics(cfg)
	require.NoError(t, err)

	server, err := NewServer(cfg, m)
	require.NoError(t, err)
	require.Equal(t, "metrics-server", server.Name())
	require.NoError(t, server.Shutdown(context.Background()))

	disabledServer, err := NewServer(Config{Enabled: false}, nil)
	require.NoError(t, err)
	require.NoError(t, disabledServer.Start())

	component, err := New(cfg)
	require.NoError(t, err)
	require.Equal(t, "metrics-server", component.Name())
	require.True(t, component.Enabled())
	require.Equal(t, "svc", component.Namespace())
	require.NotNil(t, component.Registry())

	collector := prometheus.NewCounter(prometheus.CounterOpts{Name: "another_counter_total", Help: "another"})
	require.NoError(t, component.Register(collector))
	require.NotPanics(t, func() { component.MustRegister(prometheus.NewGauge(prometheus.GaugeOpts{Name: "a_gauge", Help: "g"})) })
	require.NoError(t, component.Shutdown(context.Background()))

	var nilComponent *Component
	require.NoError(t, nilComponent.Start())
	require.NoError(t, nilComponent.Shutdown(context.Background()))
	require.Nil(t, nilComponent.Registry())
	require.Empty(t, nilComponent.Namespace())
	require.False(t, nilComponent.Enabled())
	require.NoError(t, nilComponent.Register(collector))
}

func TestHealthzHandler(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	healthzHandler(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "ok", recorder.Body.String())
}

var _ http.Handler = promhttp.Handler()
