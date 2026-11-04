package producer

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/krackl1n/careeria/kafka"
)

type MessageIDGenerator interface {
	Generate() string
}

type MessageIDGeneratorFunc func() string

func (f MessageIDGeneratorFunc) Generate() string {
	return f()
}

type CorrelationConfig struct {
	MessageIDHeader     string
	CorrelationIDHeader string
	CausationIDHeader   string

	Generator MessageIDGenerator
}

func Correlation(cfg CorrelationConfig) kafka.ProducerMiddleware {
	cfg = normalizeCorrelationConfig(cfg)

	return func(next kafka.Publisher) kafka.Publisher {
		if next == nil {
			panic("kafka producer correlation middleware: nil publisher")
		}

		return kafka.PublisherFunc(func(ctx context.Context, message kafka.Message) error {
			message = kafka.CloneMessage(message)

			messageID, _ := kafka.HeaderString(message.Headers, cfg.MessageIDHeader)
			if messageID == "" {
				messageID = cfg.Generator.Generate()
			}

			correlationID, _ := kafka.HeaderString(message.Headers, cfg.CorrelationIDHeader)
			if correlationID == "" {
				correlationID, _ = ctxmeta.CorrelationID(ctx)
			}
			if correlationID == "" {
				correlationID = messageID
			}

			causationID, _ := kafka.HeaderString(message.Headers, cfg.CausationIDHeader)
			if causationID == "" {
				causationID, _ = ctxmeta.MessageID(ctx)
			}

			if messageID != "" {
				message.Headers = kafka.SetHeader(
					message.Headers,
					cfg.MessageIDHeader,
					[]byte(messageID),
				)
				ctx = ctxmeta.WithMessageID(ctx, messageID)
			}

			if correlationID != "" {
				message.Headers = kafka.SetHeader(
					message.Headers,
					cfg.CorrelationIDHeader,
					[]byte(correlationID),
				)
				ctx = ctxmeta.WithCorrelationID(ctx, correlationID)
			}

			if causationID != "" {
				message.Headers = kafka.SetHeader(
					message.Headers,
					cfg.CausationIDHeader,
					[]byte(causationID),
				)
				ctx = ctxmeta.WithCausationID(ctx, causationID)
			}

			return next.Publish(ctx, message)
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

	if cfg.Generator == nil {
		cfg.Generator = MessageIDGeneratorFunc(generateMessageID)
	}

	return cfg
}

func generateMessageID() string {
	var value [16]byte

	if _, err := rand.Read(value[:]); err != nil {
		return ""
	}

	return hex.EncodeToString(value[:])
}
