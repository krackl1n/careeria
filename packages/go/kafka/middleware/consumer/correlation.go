package consumer

import (
	"context"
	"fmt"

	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/krackl1n/careeria/kafka"
)

type CorrelationConfig struct {
	MessageIDHeader     string
	CorrelationIDHeader string
	CausationIDHeader   string
}

func Correlation(cfg CorrelationConfig) kafka.ConsumerMiddleware {
	cfg = normalizeCorrelationConfig(cfg)

	return func(next kafka.Handler) kafka.Handler {
		if next == nil {
			panic("kafka consumer correlation middleware: nil handler")
		}

		return kafka.HandlerFunc(func(ctx context.Context, message kafka.ConsumedMessage) error {
			messageID, _ := kafka.HeaderString(message.Headers, cfg.MessageIDHeader)
			if messageID == "" {
				messageID = fallbackMessageID(message)
			}

			correlationID, _ := kafka.HeaderString(message.Headers, cfg.CorrelationIDHeader)
			if correlationID == "" {
				correlationID = messageID
			}

			causationID, _ := kafka.HeaderString(message.Headers, cfg.CausationIDHeader)

			ctx = ctxmeta.WithMessageID(ctx, messageID)
			ctx = ctxmeta.WithCorrelationID(ctx, correlationID)

			if causationID != "" {
				ctx = ctxmeta.WithCausationID(ctx, causationID)
			}

			return next.Handle(ctx, message)
		})
	}
}

func normalizeCorrelationConfig(cfg CorrelationConfig) CorrelationConfig {
	if cfg.MessageIDHeader == "" {
		cfg.MessageIDHeader = kafka.HeaderMessageID
	}

	if cfg.CorrelationIDHeader == "" {
		cfg.CorrelationIDHeader = kafka.HeaderCorrelationID
	}

	if cfg.CausationIDHeader == "" {
		cfg.CausationIDHeader = kafka.HeaderCausationID
	}

	return cfg
}

func fallbackMessageID(message kafka.ConsumedMessage) string {
	return fmt.Sprintf("%s:%d:%d", message.Topic, message.Partition, message.Offset)
}
