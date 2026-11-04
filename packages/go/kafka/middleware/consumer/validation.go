package consumer

import (
	"context"
	"fmt"

	"github.com/krackl1n/careeria/kafka"
)

type EnvelopeValidator interface {
	Validate(message kafka.ConsumedMessage) error
}

type EnvelopeValidatorFunc func(message kafka.ConsumedMessage) error

func (f EnvelopeValidatorFunc) Validate(message kafka.ConsumedMessage) error {
	return f(message)
}

type ValidationConfig struct {
	Validator EnvelopeValidator
}

func Validation(cfg ValidationConfig) kafka.ConsumerMiddleware {
	if cfg.Validator == nil {
		panic("kafka consumer validation middleware: nil validator")
	}

	return func(next kafka.Handler) kafka.Handler {
		if next == nil {
			panic("kafka consumer validation middleware: nil handler")
		}

		return kafka.HandlerFunc(func(ctx context.Context, message kafka.ConsumedMessage) error {
			if err := cfg.Validator.Validate(message); err != nil {
				return fmt.Errorf("validate kafka message: %w", err)
			}

			return next.Handle(ctx, message)
		})
	}
}
