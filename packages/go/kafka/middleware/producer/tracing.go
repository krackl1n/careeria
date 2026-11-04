package producer

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

const defaultInstrumentationName = "dev-toolbox/shared/kafka/producer"

var errPublisherPanic = errors.New("kafka producer publisher panic")

type TracingConfig struct {
	Tracer     trace.Tracer
	Propagator propagation.TextMapPropagator
}

func Tracing(cfg TracingConfig) kafka.ProducerMiddleware {
	if cfg.Tracer == nil {
		cfg.Tracer = otel.Tracer(defaultInstrumentationName)
	}

	if cfg.Propagator == nil {
		cfg.Propagator = otel.GetTextMapPropagator()
	}

	return func(next kafka.Publisher) kafka.Publisher {
		if next == nil {
			panic("kafka producer tracing middleware: nil publisher")
		}

		return kafka.PublisherFunc(func(ctx context.Context, message kafka.Message) (err error) {
			message = kafka.CloneMessage(message)

			ctx, span := cfg.Tracer.Start(
				ctx,
				"send "+message.Topic,
				trace.WithSpanKind(trace.SpanKindProducer),
				trace.WithAttributes(
					attribute.String("messaging.system", "kafka"),
					attribute.String("messaging.operation.name", "send"),
					attribute.String("messaging.destination.name", message.Topic),
					attribute.Int("messaging.message.body.size", len(message.Value)),
				),
			)

			carrier := newHeaderCarrier(&message.Headers)
			cfg.Propagator.Inject(ctx, carrier)

			panicked := true

			defer func() {
				switch {
				case panicked:
					span.RecordError(errPublisherPanic)
					span.SetStatus(otelcodes.Error, "panic")

				case err != nil:
					span.RecordError(err)
					span.SetStatus(otelcodes.Error, err.Error())
				}

				span.End()
			}()

			err = next.Publish(ctx, message)
			panicked = false

			return err
		})
	}
}

type headerCarrier struct {
	headers *[]kafka.Header
}

func newHeaderCarrier(headers *[]kafka.Header) headerCarrier {
	return headerCarrier{headers: headers}
}

func (c headerCarrier) Get(key string) string {
	if c.headers == nil {
		return ""
	}

	for _, header := range *c.headers {
		if strings.EqualFold(header.Key, key) {
			return string(header.Value)
		}
	}

	return ""
}

func (c headerCarrier) Set(key, value string) {
	if c.headers == nil {
		return
	}

	*c.headers = kafka.SetHeader(
		*c.headers,
		strings.ToLower(key),
		[]byte(value),
	)
}

func (c headerCarrier) Keys() []string {
	if c.headers == nil {
		return nil
	}

	keys := make([]string, 0, len(*c.headers))

	for _, header := range *c.headers {
		keys = append(keys, header.Key)
	}

	return keys
}
