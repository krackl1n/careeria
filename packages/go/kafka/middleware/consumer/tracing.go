package consumer

import (
	"context"
	"errors"
	"strings"

	"github.com/krackl1n/careeria/kafka"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const defaultInstrumentationName = "dev-toolbox/shared/kafka/consumer"

var errHandlerPanic = errors.New("kafka consumer handler panic")

type TracingConfig struct {
	Tracer     trace.Tracer
	Propagator propagation.TextMapPropagator
}

func Tracing(cfg TracingConfig) kafka.ConsumerMiddleware {
	if cfg.Tracer == nil {
		cfg.Tracer = otel.Tracer(defaultInstrumentationName)
	}

	if cfg.Propagator == nil {
		cfg.Propagator = otel.GetTextMapPropagator()
	}

	return func(next kafka.Handler) kafka.Handler {
		if next == nil {
			panic("kafka consumer tracing middleware: nil handler")
		}

		return kafka.HandlerFunc(func(ctx context.Context, message kafka.ConsumedMessage) (err error) {
			ctx = cfg.Propagator.Extract(ctx, newHeaderCarrier(message.Headers))

			ctx, span := cfg.Tracer.Start(
				ctx,
				"process "+message.Topic,
				trace.WithSpanKind(trace.SpanKindConsumer),
				trace.WithAttributes(
					attribute.String("messaging.system", "kafka"),
					attribute.String("messaging.operation.name", "process"),
					attribute.String("messaging.destination.name", message.Topic),
					attribute.Int("messaging.kafka.partition", int(message.Partition)),
					attribute.Int64("messaging.kafka.offset", message.Offset),
					attribute.Int("messaging.message.body.size", len(message.Value)),
				),
			)

			panicked := true

			defer func() {
				switch {
				case panicked:
					span.RecordError(errHandlerPanic)
					span.SetStatus(otelcodes.Error, "panic")

				case err != nil:
					span.RecordError(err)
					span.SetStatus(otelcodes.Error, err.Error())
				}

				span.End()
			}()

			err = next.Handle(ctx, message)
			panicked = false

			return err
		})
	}
}

type headerCarrier map[string]string

func newHeaderCarrier(headers []kafka.Header) headerCarrier {
	carrier := make(headerCarrier, len(headers))

	for _, header := range headers {
		carrier[strings.ToLower(header.Key)] = string(header.Value)
	}

	return carrier
}

func (c headerCarrier) Get(key string) string {
	return c[strings.ToLower(key)]
}

func (c headerCarrier) Set(key, value string) {
	c[strings.ToLower(key)] = value
}

func (c headerCarrier) Keys() []string {
	keys := make([]string, 0, len(c))

	for key := range c {
		keys = append(keys, key)
	}

	return keys
}
