package preset

import (
	"log/slog"

	"github.com/krackl1n/careeria/kafka"
	producermw "github.com/krackl1n/careeria/kafka/middleware/producer"
)

type ProducerConfig struct {
	Logger   *slog.Logger
	Producer string

	Correlation producermw.CorrelationConfig
	Tracing     producermw.TracingConfig
	Metrics     *producermw.Metrics

	Timeout    *producermw.TimeoutConfig
	Validation producermw.ValidationConfig
}

func Producer(cfg ProducerConfig) []kafka.ProducerMiddleware {
	if cfg.Producer == "" {
		panic("kafka producer preset: producer is required")
	}

	if cfg.Metrics == nil {
		panic("kafka producer preset: nil metrics")
	}

	if cfg.Validation.Validator == nil {
		cfg.Validation.Validator = producermw.DefaultEnvelopeValidator{}
	}

	middlewares := []kafka.ProducerMiddleware{
		producermw.Correlation(cfg.Correlation),
		producermw.Tracing(cfg.Tracing),
		producermw.Logging(producermw.LoggingConfig{
			Logger:   cfg.Logger,
			Producer: cfg.Producer,
		}),
		cfg.Metrics.Middleware(),
	}

	if cfg.Timeout != nil {
		middlewares = append(middlewares, producermw.Timeout(*cfg.Timeout))
	}

	middlewares = append(middlewares, producermw.Validation(cfg.Validation))

	return middlewares
}
