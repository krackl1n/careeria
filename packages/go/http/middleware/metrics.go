package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/prometheus/client_golang/prometheus"
)

type MetricsConfig struct {
	Namespace string
	Subsystem string

	DurationBuckets []float64
}

func Metrics(config MetricsConfig, registerer prometheus.Registerer) func(http.Handler) http.Handler {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}

	durationBuckets := config.DurationBuckets
	if len(durationBuckets) == 0 {
		durationBuckets = prometheus.DefBuckets
	}

	requestsTotal := registerCounterVec(registerer, prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests.",
		},
		[]string{"method", "status"},
	))

	requestDuration := registerHistogramVec(registerer, prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds.",
			Buckets:   durationBuckets,
		},
		[]string{"method", "status"},
	))

	requestsInFlight := registerGauge(registerer, prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "http_requests_in_flight",
			Help:      "Current number of HTTP requests being processed.",
		},
	))

	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("metrics middleware: nil handler")
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()

			var captured httpsnoop.Metrics
			panicked := true

			requestsInFlight.Inc()

			defer func() {
				requestsInFlight.Dec()

				status := captured.Code
				if status == 0 {
					status = http.StatusOK
				}

				if panicked {
					status = http.StatusInternalServerError
				}

				method := normalizeMethod(r.Method)
				statusLabel := strconv.Itoa(status)
				duration := time.Since(startedAt)

				requestsTotal.WithLabelValues(method, statusLabel).Inc()
				requestDuration.WithLabelValues(method, statusLabel).Observe(duration.Seconds())
			}()

			captured = httpsnoop.CaptureMetrics(next, w, r)
			panicked = false
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

		panic("metrics middleware: register counter: " + err.Error())
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

		panic("metrics middleware: register histogram: " + err.Error())
	}

	return collector
}

func registerGauge(registerer prometheus.Registerer, collector prometheus.Gauge) prometheus.Gauge {
	if err := registerer.Register(collector); err != nil {
		if registered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := registered.ExistingCollector.(prometheus.Gauge); ok {
				return existing
			}
		}

		panic("metrics middleware: register gauge: " + err.Error())
	}

	return collector
}
