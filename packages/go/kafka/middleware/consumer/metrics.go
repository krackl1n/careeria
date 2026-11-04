package consumer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/krackl1n/careeria/kafka"

	"github.com/prometheus/client_golang/prometheus"
)

type MetricsConfig struct {
	Registerer prometheus.Registerer

	Namespace string
	Subsystem string

	ConsumerGroup string

	DurationBuckets []float64
}

type Metrics struct {
	consumerGroup string

	started  *prometheus.CounterVec
	handled  *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight *prometheus.GaugeVec
}

func NewMetrics(cfg MetricsConfig) (*Metrics, error) {
	if cfg.ConsumerGroup == "" {
		return nil, errors.New("kafka consumer metrics: consumer group is required")
	}

	registerer := cfg.Registerer
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}

	buckets := cfg.DurationBuckets
	if len(buckets) == 0 {
		buckets = prometheus.DefBuckets
	}

	started, err := registerCounterVec(
		registerer,
		prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "messages_started_total",
				Help:      "Total number of Kafka messages whose processing started.",
			},
			[]string{"consumer_group", "topic"},
		),
	)
	if err != nil {
		return nil, err
	}

	handled, err := registerCounterVec(
		registerer,
		prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "messages_handled_total",
				Help:      "Total number of processed Kafka messages.",
			},
			[]string{"consumer_group", "topic", "result"},
		),
	)
	if err != nil {
		return nil, err
	}

	duration, err := registerHistogramVec(
		registerer,
		prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "message_duration_seconds",
				Help:      "Kafka message processing duration in seconds.",
				Buckets:   buckets,
			},
			[]string{"consumer_group", "topic", "result"},
		),
	)
	if err != nil {
		return nil, err
	}

	inFlight, err := registerGaugeVec(
		registerer,
		prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "messages_in_flight",
				Help:      "Current number of Kafka messages being processed.",
			},
			[]string{"consumer_group", "topic"},
		),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		consumerGroup: cfg.ConsumerGroup,
		started:       started,
		handled:       handled,
		duration:      duration,
		inFlight:      inFlight,
	}, nil
}

func (m *Metrics) Middleware() kafka.ConsumerMiddleware {
	if m == nil {
		panic("kafka consumer metrics middleware: nil metrics")
	}

	return func(next kafka.Handler) kafka.Handler {
		if next == nil {
			panic("kafka consumer metrics middleware: nil handler")
		}

		return kafka.HandlerFunc(func(ctx context.Context, message kafka.ConsumedMessage) (err error) {
			m.started.WithLabelValues(m.consumerGroup, message.Topic).Inc()
			m.inFlight.WithLabelValues(m.consumerGroup, message.Topic).Inc()

			startedAt := time.Now()
			panicked := true

			defer func() {
				result := processingResult(err, panicked)

				m.inFlight.WithLabelValues(m.consumerGroup, message.Topic).Dec()
				m.handled.WithLabelValues(
					m.consumerGroup,
					message.Topic,
					result,
				).Inc()
				m.duration.WithLabelValues(
					m.consumerGroup,
					message.Topic,
					result,
				).Observe(time.Since(startedAt).Seconds())
			}()

			err = next.Handle(ctx, message)
			panicked = false

			return err
		})
	}
}

func registerCounterVec(
	registerer prometheus.Registerer,
	collector *prometheus.CounterVec,
) (*prometheus.CounterVec, error) {
	if err := registerer.Register(collector); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError

		if errors.As(err, &alreadyRegistered) {
			existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.CounterVec)
			if ok {
				return existing, nil
			}
		}

		return nil, fmt.Errorf("register kafka consumer counter: %w", err)
	}

	return collector, nil
}

func registerHistogramVec(
	registerer prometheus.Registerer,
	collector *prometheus.HistogramVec,
) (*prometheus.HistogramVec, error) {
	if err := registerer.Register(collector); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError

		if errors.As(err, &alreadyRegistered) {
			existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.HistogramVec)
			if ok {
				return existing, nil
			}
		}

		return nil, fmt.Errorf("register kafka consumer histogram: %w", err)
	}

	return collector, nil
}

func registerGaugeVec(
	registerer prometheus.Registerer,
	collector *prometheus.GaugeVec,
) (*prometheus.GaugeVec, error) {
	if err := registerer.Register(collector); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError

		if errors.As(err, &alreadyRegistered) {
			existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.GaugeVec)
			if ok {
				return existing, nil
			}
		}

		return nil, fmt.Errorf("register kafka consumer gauge: %w", err)
	}

	return collector, nil
}
