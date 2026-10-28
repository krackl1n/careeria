package producer

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/krackl1n/careeria/kafka"
)

const (
	resultSuccess  = "success"
	resultError    = "error"
	resultPanic    = "panic"
	resultCanceled = "canceled"
	resultTimeout  = "timeout"
)

type LoggingConfig struct {
	Logger   *slog.Logger
	Producer string
}

func Logging(cfg LoggingConfig) kafka.ProducerMiddleware {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return func(next kafka.Publisher) kafka.Publisher {
		if next == nil {
			panic("kafka producer logging middleware: nil publisher")
		}

		return kafka.PublisherFunc(func(ctx context.Context, message kafka.Message) (err error) {
			startedAt := time.Now()
			panicked := true

			defer func() {
				duration := time.Since(startedAt)
				result := publishResult(err, panicked)

				messageID, _ := ctxmeta.MessageID(ctx)
				if messageID == "" {
					messageID, _ = kafka.HeaderString(message.Headers, kafka.HeaderMessageID)
				}

				correlationID, _ := ctxmeta.CorrelationID(ctx)
				if correlationID == "" {
					correlationID, _ = kafka.HeaderString(
						message.Headers,
						kafka.HeaderCorrelationID,
					)
				}

				causationID, _ := ctxmeta.CausationID(ctx)
				if causationID == "" {
					causationID, _ = kafka.HeaderString(
						message.Headers,
						kafka.HeaderCausationID,
					)
				}

				args := []any{
					"producer", cfg.Producer,
					"topic", message.Topic,
					"message_id", messageID,
					"correlation_id", correlationID,
					"causation_id", causationID,
					"key_bytes", len(message.Key),
					"value_bytes", len(message.Value),
					"result", result,
					"duration", duration.String(),
					"duration_ms", float64(duration.Microseconds()) / 1000,
				}

				if err != nil {
					args = append(args, slog.Any("error", err))
				}

				switch result {
				case resultSuccess:
					cfg.Logger.InfoContext(ctx, "kafka message published", args...)

				case resultCanceled, resultTimeout:
					cfg.Logger.WarnContext(
						ctx,
						"kafka message publishing interrupted",
						args...,
					)

				default:
					cfg.Logger.ErrorContext(
						ctx,
						"kafka message publishing failed",
						args...,
					)
				}
			}()

			err = next.Publish(ctx, message)
			panicked = false

			return err
		})
	}
}

func publishResult(err error, panicked bool) string {
	switch {
	case panicked:
		return resultPanic
	case err == nil:
		return resultSuccess
	case errors.Is(err, context.Canceled):
		return resultCanceled
	case errors.Is(err, context.DeadlineExceeded):
		return resultTimeout
	default:
		return resultError
	}
}
