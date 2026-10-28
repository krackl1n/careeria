package interceptor

import (
	"context"
	"fmt"
	"time"

	"github.com/krackl1n/careeria/grpc/matcher"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MetricsConfig struct {
	Namespace string
	Subsystem string

	DurationBuckets []float64
}

type Metrics struct {
	started  *prometheus.CounterVec
	handled  *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight *prometheus.GaugeVec
}

func NewMetrics(cfg MetricsConfig, registerer prometheus.Registerer) (*Metrics, error) {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}

	buckets := cfg.DurationBuckets
	if len(buckets) == 0 {
		buckets = prometheus.DefBuckets
	}

	metrics := &Metrics{
		started: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "requests_started_total",
				Help:      "Total number of started gRPC requests.",
			},
			[]string{"service", "method", "type"},
		),
		handled: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "requests_handled_total",
				Help:      "Total number of completed gRPC requests.",
			},
			[]string{"service", "method", "type", "code"},
		),
		duration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "request_duration_seconds",
				Help:      "Duration of gRPC requests.",
				Buckets:   buckets,
			},
			[]string{"service", "method", "type", "code"},
		),
		inFlight: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "requests_in_flight",
				Help:      "Number of in-flight gRPC requests.",
			},
			[]string{"service", "method", "type"},
		),
	}

	collectors := []prometheus.Collector{
		metrics.started,
		metrics.handled,
		metrics.duration,
		metrics.inFlight,
	}

	for _, collector := range collectors {
		if err := registerer.Register(collector); err != nil {
			return nil, fmt.Errorf("register grpc metrics: %w", err)
		}
	}

	return metrics, nil
}

func (m *Metrics) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	if m == nil {
		panic("grpc metrics interceptor: nil metrics")
	}

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		service, method := matcher.SplitFullMethod(info.FullMethod)
		const rpcType = "unary"

		m.started.WithLabelValues(service, method, rpcType).Inc()
		m.inFlight.WithLabelValues(service, method, rpcType).Inc()

		startedAt := time.Now()
		panicked := true

		defer func() {
			code := status.Code(err)
			if panicked {
				code = codes.Internal
			}

			m.inFlight.WithLabelValues(service, method, rpcType).Dec()
			m.handled.WithLabelValues(service, method, rpcType, code.String()).Inc()
			m.duration.WithLabelValues(
				service,
				method,
				rpcType,
				code.String(),
			).Observe(time.Since(startedAt).Seconds())
		}()

		resp, err = handler(ctx, req)
		panicked = false

		return resp, err
	}
}

func (m *Metrics) StreamServerInterceptor() grpc.StreamServerInterceptor {
	if m == nil {
		panic("grpc metrics interceptor: nil metrics")
	}

	return func(
		srv any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		service, method := matcher.SplitFullMethod(info.FullMethod)
		rpcType := streamType(info)

		m.started.WithLabelValues(service, method, rpcType).Inc()
		m.inFlight.WithLabelValues(service, method, rpcType).Inc()

		startedAt := time.Now()
		panicked := true

		defer func() {
			code := status.Code(err)
			if panicked {
				code = codes.Internal
			}

			m.inFlight.WithLabelValues(service, method, rpcType).Dec()
			m.handled.WithLabelValues(service, method, rpcType, code.String()).Inc()
			m.duration.WithLabelValues(
				service,
				method,
				rpcType,
				code.String(),
			).Observe(time.Since(startedAt).Seconds())
		}()

		err = handler(srv, stream)
		panicked = false

		return err
	}
}
