package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type MetricsConfig struct {
	Registerer prometheus.Registerer

	Namespace string
	Subsystem string

	Target string

	DurationBuckets []float64
}

func Metrics(cfg MetricsConfig) func(http.RoundTripper) http.RoundTripper {
	registerer := cfg.Registerer
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}

	if cfg.Target == "" {
		cfg.Target = "unknown"
	}

	buckets := cfg.DurationBuckets
	if len(buckets) == 0 {
		buckets = prometheus.DefBuckets
	}

	requestsTotal := registerCounterVec(registerer, prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "http_client_requests_total",
			Help:      "Total number of outbound HTTP requests.",
		},
		[]string{"target", "method", "status"},
	))

	requestDuration := registerHistogramVec(registerer, prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "http_client_request_duration_seconds",
			Help:      "Outbound HTTP request duration in seconds.",
			Buckets:   buckets,
		},
		[]string{"target", "method", "status"},
	))

	return func(next http.RoundTripper) http.RoundTripper {
		if next == nil {
			next = http.DefaultTransport
		}

		return roundTripFunc(func(r *http.Request) (*http.Response, error) {
			startedAt := time.Now()

			resp, err := next.RoundTrip(r)

			status := "error"
			if resp != nil {
				status = strconv.Itoa(resp.StatusCode)
			}

			method := normalizeMethod(r.Method)
			duration := time.Since(startedAt)

			requestsTotal.WithLabelValues(cfg.Target, method, status).Inc()
			requestDuration.WithLabelValues(cfg.Target, method, status).Observe(duration.Seconds())

			return resp, err
		})
	}
}

func normalizeMethod(method string) string {
	switch method {
	case http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace:
		return method
	default:
		return "OTHER"
	}
}

func registerCounterVec(registerer prometheus.Registerer, collector *prometheus.CounterVec) *prometheus.CounterVec {
	if err := registerer.Register(collector); err != nil {
		if registered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := registered.ExistingCollector.(*prometheus.CounterVec); ok {
				return existing
			}
		}

		panic("http client metrics middleware: register counter: " + err.Error())
	}

	return collector
}

func registerHistogramVec(registerer prometheus.Registerer, collector *prometheus.HistogramVec) *prometheus.HistogramVec {
	if err := registerer.Register(collector); err != nil {
		if registered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := registered.ExistingCollector.(*prometheus.HistogramVec); ok {
				return existing
			}
		}

		panic("http client metrics middleware: register histogram: " + err.Error())
	}

	return collector
}
