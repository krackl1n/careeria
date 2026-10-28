package interceptor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/krackl1n/careeria/grpc/matcher"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
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

	if cfg.Subsystem == "" {
		cfg.Subsystem = "grpc_client"
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
				Help:      "Total number of started gRPC client requests.",
			},
			[]string{
				"peer_service",
				"grpc_service",
				"grpc_method",
				"grpc_type",
			},
		),
		handled: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "requests_handled_total",
				Help:      "Total number of completed gRPC client requests.",
			},
			[]string{
				"peer_service",
				"grpc_service",
				"grpc_method",
				"grpc_type",
				"grpc_code",
			},
		),
		duration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "request_duration_seconds",
				Help:      "Duration of gRPC client requests.",
				Buckets:   buckets,
			},
			[]string{
				"peer_service",
				"grpc_service",
				"grpc_method",
				"grpc_type",
				"grpc_code",
			},
		),
		inFlight: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "requests_in_flight",
				Help:      "Number of in-flight gRPC client requests.",
			},
			[]string{
				"peer_service",
				"grpc_service",
				"grpc_method",
				"grpc_type",
			},
		),
	}

	var err error
	metrics.started, err = registerCounterVec(registerer, metrics.started)
	if err != nil {
		return nil, err
	}
	metrics.handled, err = registerCounterVec(registerer, metrics.handled)
	if err != nil {
		return nil, err
	}
	metrics.duration, err = registerHistogramVec(registerer, metrics.duration)
	if err != nil {
		return nil, err
	}
	metrics.inFlight, err = registerGaugeVec(registerer, metrics.inFlight)
	if err != nil {
		return nil, err
	}

	return metrics, nil
}

func registerCounterVec(registerer prometheus.Registerer, collector *prometheus.CounterVec) (*prometheus.CounterVec, error) {
	if err := registerer.Register(collector); err != nil {
		if registered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := registered.ExistingCollector.(*prometheus.CounterVec); ok {
				return existing, nil
			}
		}

		return nil, fmt.Errorf("register grpc client metrics: %w", err)
	}

	return collector, nil
}

func registerHistogramVec(registerer prometheus.Registerer, collector *prometheus.HistogramVec) (*prometheus.HistogramVec, error) {
	if err := registerer.Register(collector); err != nil {
		if registered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := registered.ExistingCollector.(*prometheus.HistogramVec); ok {
				return existing, nil
			}
		}

		return nil, fmt.Errorf("register grpc client metrics: %w", err)
	}

	return collector, nil
}

func registerGaugeVec(registerer prometheus.Registerer, collector *prometheus.GaugeVec) (*prometheus.GaugeVec, error) {
	if err := registerer.Register(collector); err != nil {
		if registered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := registered.ExistingCollector.(*prometheus.GaugeVec); ok {
				return existing, nil
			}
		}

		return nil, fmt.Errorf("register grpc client metrics: %w", err)
	}

	return collector, nil
}

func (m *Metrics) UnaryClientInterceptor(peerService string) grpc.UnaryClientInterceptor {
	if m == nil {
		panic("grpc client metrics interceptor: nil metrics")
	}

	peerService = normalizePeerService(peerService)

	return func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		service, rpcMethod := matcher.SplitFullMethod(method)
		const rpcType = "unary"

		m.started.WithLabelValues(
			peerService,
			service,
			rpcMethod,
			rpcType,
		).Inc()

		m.inFlight.WithLabelValues(
			peerService,
			service,
			rpcMethod,
			rpcType,
		).Inc()

		startedAt := time.Now()

		err := invoker(ctx, method, req, reply, cc, opts...)

		m.complete(
			peerService,
			service,
			rpcMethod,
			rpcType,
			startedAt,
			err,
		)

		return err
	}
}

func (m *Metrics) StreamClientInterceptor(peerService string) grpc.StreamClientInterceptor {
	if m == nil {
		panic("grpc client metrics interceptor: nil metrics")
	}

	peerService = normalizePeerService(peerService)

	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		service, rpcMethod := matcher.SplitFullMethod(method)
		rpcType := streamType(desc)

		m.started.WithLabelValues(
			peerService,
			service,
			rpcMethod,
			rpcType,
		).Inc()

		m.inFlight.WithLabelValues(
			peerService,
			service,
			rpcMethod,
			rpcType,
		).Inc()

		startedAt := time.Now()

		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			m.complete(
				peerService,
				service,
				rpcMethod,
				rpcType,
				startedAt,
				err,
			)

			return nil, err
		}

		return wrapClientStream(stream, ctx, desc, func(err error) {
			m.complete(
				peerService,
				service,
				rpcMethod,
				rpcType,
				startedAt,
				err,
			)
		}), nil
	}
}

func (m *Metrics) complete(
	peerService string,
	service string,
	method string,
	rpcType string,
	startedAt time.Time,
	err error,
) {
	code := codeFromError(err).String()

	m.inFlight.WithLabelValues(
		peerService,
		service,
		method,
		rpcType,
	).Dec()

	m.handled.WithLabelValues(
		peerService,
		service,
		method,
		rpcType,
		code,
	).Inc()

	m.duration.WithLabelValues(
		peerService,
		service,
		method,
		rpcType,
		code,
	).Observe(time.Since(startedAt).Seconds())
}

func normalizePeerService(peerService string) string {
	peerService = strings.TrimSpace(peerService)
	if peerService == "" {
		return "unknown"
	}

	return peerService
}
