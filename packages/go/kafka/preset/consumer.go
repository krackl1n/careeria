package preset

import (
	"log/slog"

	"github.com/krackl1n/careeria/kafka"
	consumermw "github.com/krackl1n/careeria/kafka/middleware/consumer"
)

type ConsumerConfig struct {
	Logger        *slog.Logger
	ConsumerGroup string

	Correlation consumermw.CorrelationConfig
	Tracing     consumermw.TracingConfig
	Metrics     *consumermw.Metrics

	Timeout    *consumermw.TimeoutConfig
	Validation *consumermw.ValidationConfig

	PanicHandler consumermw.PanicHandler
}

func Consumer(cfg ConsumerConfig) []kafka.ConsumerMiddleware {
	if cfg.ConsumerGroup == "" {
		panic("kafka consumer preset: consumer group is required")
	}

	if cfg.Metrics == nil {
		panic("kafka consumer preset: nil metrics")
	}

	middlewares := []kafka.ConsumerMiddleware{
		consumermw.Recovery(cfg.Logger, cfg.PanicHandler),
		consumermw.Correlation(cfg.Correlation),
		consumermw.Tracing(cfg.Tracing),
		consumermw.Logging(consumermw.LoggingConfig{
			Logger:        cfg.Logger,
			ConsumerGroup: cfg.ConsumerGroup,
		}),
		cfg.Metrics.Middleware(),
	}

	timeout := consumermw.TimeoutConfig{Provider: consumermw.StaticTimeout(kafka.DefaultConsumerProcessingTimeout)}
	if cfg.Timeout != nil {
		timeout = *cfg.Timeout
	}
	middlewares = append(middlewares, consumermw.Timeout(timeout))

	if cfg.Validation != nil {
		middlewares = append(middlewares, consumermw.Validation(*cfg.Validation))
	}

	return middlewares
}
