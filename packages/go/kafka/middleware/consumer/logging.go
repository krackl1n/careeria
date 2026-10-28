package consumer

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
	Logger        *slog.Logger
	ConsumerGroup string
}

func Logging(cfg LoggingConfig) kafka.ConsumerMiddleware {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return func(next kafka.Handler) kafka.Handler {
		if next == nil {
			panic("kafka consumer logging middleware: nil handler")
		}

		return kafka.HandlerFunc(func(ctx context.Context, message kafka.ConsumedMessage) (err error) {
			startedAt := time.Now()
			panicked := true

			defer func() {
				duration := time.Since(startedAt)
				result := processingResult(err, panicked)
				messageID, _ := ctxmeta.MessageID(ctx)
				correlationID, _ := ctxmeta.CorrelationID(ctx)
				causationID, _ := ctxmeta.CausationID(ctx)

				args := []any{
					"consumer_group", cfg.ConsumerGroup,
					"topic", message.Topic,
					"partition", message.Partition,
					"offset", message.Offset,
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
					cfg.Logger.InfoContext(ctx, "kafka message processed", args...)

				case resultCanceled, resultTimeout:
					cfg.Logger.WarnContext(
						ctx,
						"kafka message processing interrupted",
						args...,
					)

				default:
					cfg.Logger.ErrorContext(
						ctx,
						"kafka message processing failed",
						args...,
					)
				}
			}()

			err = next.Handle(ctx, message)
			panicked = false

			return err
		})
	}
}

func processingResult(err error, panicked bool) string {
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
