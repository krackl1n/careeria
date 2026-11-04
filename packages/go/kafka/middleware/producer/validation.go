package producer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/krackl1n/careeria/kafka"
)

type EnvelopeValidator interface {
	Validate(message kafka.Message) error
}

type EnvelopeValidatorFunc func(message kafka.Message) error

func (f EnvelopeValidatorFunc) Validate(message kafka.Message) error {
	return f(message)
}

type ValidationConfig struct {
	Validator EnvelopeValidator
}

func Validation(cfg ValidationConfig) kafka.ProducerMiddleware {
	if cfg.Validator == nil {
		panic("kafka producer validation middleware: nil validator")
	}

	return func(next kafka.Publisher) kafka.Publisher {
		if next == nil {
			panic("kafka producer validation middleware: nil publisher")
		}

		return kafka.PublisherFunc(func(ctx context.Context, message kafka.Message) error {
			if err := cfg.Validator.Validate(message); err != nil {
				return fmt.Errorf("validate kafka message: %w", err)
			}

			return next.Publish(ctx, message)
		})
	}
}

type DefaultEnvelopeValidator struct {
	RequireValue bool

	MaxKeyBytes    int
	MaxValueBytes  int
	MaxHeaders     int
	MaxHeaderBytes int
}

func (v DefaultEnvelopeValidator) Validate(message kafka.Message) error {
	if strings.TrimSpace(message.Topic) == "" {
		return errors.New("topic is required")
	}

	if v.RequireValue && message.Value == nil {
		return errors.New("value is required")
	}

	if v.MaxKeyBytes > 0 && len(message.Key) > v.MaxKeyBytes {
		return errors.New("key exceeds maximum size")
	}

	if v.MaxValueBytes > 0 && len(message.Value) > v.MaxValueBytes {
		return errors.New("value exceeds maximum size")
	}

	if v.MaxHeaders > 0 && len(message.Headers) > v.MaxHeaders {
		return errors.New("headers exceed maximum count")
	}

	headerBytes := 0

	for _, header := range message.Headers {
		if strings.TrimSpace(header.Key) == "" {
			return errors.New("header key is required")
		}

		headerBytes += len(header.Key) + len(header.Value)
	}

	if v.MaxHeaderBytes > 0 && headerBytes > v.MaxHeaderBytes {
		return errors.New("headers exceed maximum size")
	}

	return nil
}
