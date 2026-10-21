package producer

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/krackl1n/careeria/kafka"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func contextMetadata(
	t *testing.T,
	ctx context.Context,
	getter func(context.Context) (string, bool),
) string {
	t.Helper()
	value, _ := getter(ctx)
	return value
}

func TestCorrelation(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil publisher", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "kafka producer correlation middleware: nil publisher", func() {
			Correlation(CorrelationConfig{})(nil)
		})
	})

	t.Run("sets headers and context values", func(t *testing.T) {
		t.Parallel()

		original := testMessage()
		ctx := ctxmeta.WithMessageID(context.Background(), "parent-msg")

		err := Correlation(CorrelationConfig{
			Generator: MessageIDGeneratorFunc(func() string {
				return "msg-123"
			}),
		})(kafka.PublisherFunc(func(ctx context.Context, message kafka.Message) error {
			assert.Equal(t, "msg-123", headerString(t, message.Headers, kafka.HeaderMessageID))
			assert.Equal(t, "msg-123", headerString(t, message.Headers, kafka.HeaderCorrelationID))
			assert.Equal(t, "parent-msg", headerString(t, message.Headers, kafka.HeaderCausationID))
			assert.Equal(t, "msg-123", contextMetadata(t, ctx, ctxmeta.MessageID))
			assert.Equal(t, "msg-123", contextMetadata(t, ctx, ctxmeta.CorrelationID))
			assert.Equal(t, "parent-msg", contextMetadata(t, ctx, ctxmeta.CausationID))
			return nil
		})).Publish(ctx, original)

		require.NoError(t, err)
		assert.Empty(t, original.Headers)
	})

	t.Run("preserves existing headers", func(t *testing.T) {
		t.Parallel()

		message := testMessage()
		message.Headers = []kafka.Header{
			{Key: kafka.HeaderMessageID, Value: []byte("existing-msg")},
			{Key: kafka.HeaderCorrelationID, Value: []byte("existing-corr")},
			{Key: kafka.HeaderCausationID, Value: []byte("existing-cause")},
		}

		err := Correlation(CorrelationConfig{})(kafka.PublisherFunc(func(ctx context.Context, message kafka.Message) error {
			assert.Equal(t, "existing-msg", contextMetadata(t, ctx, ctxmeta.MessageID))
			assert.Equal(t, "existing-corr", contextMetadata(t, ctx, ctxmeta.CorrelationID))
			assert.Equal(t, "existing-cause", contextMetadata(t, ctx, ctxmeta.CausationID))
			return nil
		})).Publish(context.Background(), message)

		require.NoError(t, err)
	})
}

func TestTimeout(t *testing.T) {
	t.Parallel()

	t.Run("panics on invalid config", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "kafka producer timeout middleware: nil provider", func() {
			Timeout(TimeoutConfig{})
		})
		require.PanicsWithValue(t, "kafka producer timeout middleware: nil publisher", func() {
			Timeout(TimeoutConfig{Provider: StaticTimeout(time.Second)})(nil)
		})
	})

	t.Run("returns existing context error", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := Timeout(TimeoutConfig{Provider: StaticTimeout(time.Second)})(kafka.PublisherFunc(func(context.Context, kafka.Message) error {
			t.Fatal("next must not be called")
			return nil
		})).Publish(ctx, testMessage())

		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("applies timeout", func(t *testing.T) {
		t.Parallel()

		err := Timeout(TimeoutConfig{Provider: StaticTimeout(time.Minute)})(kafka.PublisherFunc(func(ctx context.Context, message kafka.Message) error {
			deadline, ok := ctx.Deadline()
			require.True(t, ok)
			assert.WithinDuration(t, time.Now().Add(time.Minute), deadline, 2*time.Second)
			return nil
		})).Publish(context.Background(), testMessage())

		require.NoError(t, err)
	})
}

func TestValidation(t *testing.T) {
	t.Parallel()

	t.Run("panics on invalid config", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "kafka producer validation middleware: nil validator", func() {
			Validation(ValidationConfig{})
		})
		require.PanicsWithValue(t, "kafka producer validation middleware: nil publisher", func() {
			Validation(ValidationConfig{Validator: DefaultEnvelopeValidator{}})(nil)
		})
	})

	t.Run("returns validation error", func(t *testing.T) {
		t.Parallel()

		err := Validation(ValidationConfig{
			Validator: DefaultEnvelopeValidator{RequireValue: true},
		})(kafka.PublisherFunc(func(context.Context, kafka.Message) error {
			t.Fatal("next must not be called")
			return nil
		})).Publish(context.Background(), kafka.Message{Topic: "orders"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "validate kafka message")
	})

	t.Run("default validator checks envelope limits", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			message kafka.Message
			cfg     DefaultEnvelopeValidator
			wantErr string
		}{
			{name: "missing topic", message: kafka.Message{}, wantErr: "topic is required"},
			{name: "missing value", message: kafka.Message{Topic: "orders"}, cfg: DefaultEnvelopeValidator{RequireValue: true}, wantErr: "value is required"},
			{name: "key too large", message: kafka.Message{Topic: "orders", Key: []byte("12")}, cfg: DefaultEnvelopeValidator{MaxKeyBytes: 1}, wantErr: "key exceeds maximum size"},
			{name: "value too large", message: kafka.Message{Topic: "orders", Value: []byte("12")}, cfg: DefaultEnvelopeValidator{MaxValueBytes: 1}, wantErr: "value exceeds maximum size"},
			{
				name: "too many headers",
				message: kafka.Message{
					Topic:   "orders",
					Headers: []kafka.Header{{Key: "a"}, {Key: "b"}},
				},
				cfg:     DefaultEnvelopeValidator{MaxHeaders: 1},
				wantErr: "headers exceed maximum count",
			},
			{name: "empty header key", message: kafka.Message{Topic: "orders", Headers: []kafka.Header{{Value: []byte("v")}}}, wantErr: "header key is required"},
			{name: "headers too large", message: kafka.Message{Topic: "orders", Headers: []kafka.Header{{Key: "a", Value: []byte("12")}}}, cfg: DefaultEnvelopeValidator{MaxHeaderBytes: 1}, wantErr: "headers exceed maximum size"},
			{name: "valid", message: testMessage()},
		}

		for _, tt := range tests {
			tt := tt

			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				err := tt.cfg.Validate(tt.message)
				if tt.wantErr == "" {
					require.NoError(t, err)
					return
				}

				require.EqualError(t, err, tt.wantErr)
			})
		}
	})
}

func TestLogging(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		panicValue any
		wantLevel  string
		wantResult string
	}{
		{name: "success", wantLevel: "INFO", wantResult: resultSuccess},
		{name: "error", err: errors.New("publish failed"), wantLevel: "ERROR", wantResult: resultError},
		{name: "canceled", err: context.Canceled, wantLevel: "WARN", wantResult: resultCanceled},
		{name: "timeout", err: context.DeadlineExceeded, wantLevel: "WARN", wantResult: resultTimeout},
		{name: "panic", panicValue: "boom", wantLevel: "ERROR", wantResult: resultPanic},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, nil))
			ctx := ctxmeta.WithMessageID(context.Background(), "msg-123")
			ctx = ctxmeta.WithCorrelationID(ctx, "corr-123")
			ctx = ctxmeta.WithCausationID(ctx, "cause-123")

			publisher := Logging(LoggingConfig{
				Logger:   logger,
				Producer: "orders-producer",
			})(kafka.PublisherFunc(func(context.Context, kafka.Message) error {
				if tt.panicValue != nil {
					panic(tt.panicValue)
				}
				return tt.err
			}))

			if tt.panicValue != nil {
				require.PanicsWithValue(t, tt.panicValue, func() {
					_ = publisher.Publish(ctx, testMessage())
				})
			} else {
				err := publisher.Publish(ctx, testMessage())
				if tt.err != nil {
					require.ErrorIs(t, err, tt.err)
				} else {
					require.NoError(t, err)
				}
			}

			assert.Contains(t, logs.String(), tt.wantLevel)
			assert.Contains(t, logs.String(), tt.wantResult)
			assert.Contains(t, logs.String(), "orders-producer")
			assert.Contains(t, logs.String(), "msg-123")
			assert.Contains(t, logs.String(), "corr-123")
			assert.Contains(t, logs.String(), "cause-123")
		})
	}
}

func TestMetrics(t *testing.T) {
	t.Parallel()

	t.Run("requires producer", func(t *testing.T) {
		t.Parallel()

		metrics, err := NewMetrics(MetricsConfig{})
		require.Nil(t, metrics)
		require.EqualError(t, err, "kafka producer metrics: producer is required")
	})

	t.Run("reuses collectors and records results", func(t *testing.T) {
		t.Parallel()

		registerer := prometheus.NewRegistry()
		metrics, err := NewMetrics(MetricsConfig{Registerer: registerer, Producer: "orders-producer"})
		require.NoError(t, err)

		second, err := NewMetrics(MetricsConfig{Registerer: registerer, Producer: "orders-producer"})
		require.NoError(t, err)
		assert.Same(t, metrics.started, second.started)

		publisher := metrics.Middleware()(kafka.PublisherFunc(func(ctx context.Context, message kafka.Message) error {
			if string(message.Key) == "error" {
				return io.ErrUnexpectedEOF
			}
			return nil
		}))

		require.NoError(t, publisher.Publish(context.Background(), testMessage()))

		failed := testMessage()
		failed.Key = []byte("error")
		require.ErrorIs(t, publisher.Publish(context.Background(), failed), io.ErrUnexpectedEOF)

		families, err := registerer.Gather()
		require.NoError(t, err)
		assertMetricCounterValue(t, families, "messages_started_total", 2)
		assertMetricCounterValue(t, families, "messages_handled_total", 2)
		assertMetricGaugeValue(t, families, "messages_in_flight", 0)
	})
}

func TestTracing(t *testing.T) {
	t.Parallel()

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() {
		require.NoError(t, provider.Shutdown(context.Background()))
	})

	var seen kafka.Message
	err := Tracing(TracingConfig{
		Tracer:     provider.Tracer("test"),
		Propagator: propagation.TraceContext{},
	})(kafka.PublisherFunc(func(ctx context.Context, message kafka.Message) error {
		seen = message
		return errors.New("publish failed")
	})).Publish(context.Background(), testMessage())

	require.Error(t, err)
	assert.NotEmpty(t, headerString(t, seen.Headers, "traceparent"))

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, "send orders", spans[0].Name())
	assert.Equal(t, codes.Error, spans[0].Status().Code)
}

func testMessage() kafka.Message {
	return kafka.Message{
		Topic: "orders",
		Key:   []byte("order-key"),
		Value: []byte(`{"id":"order-123"}`),
	}
}

func headerString(t *testing.T, headers []kafka.Header, key string) string {
	t.Helper()

	value, ok := kafka.HeaderString(headers, key)
	require.True(t, ok)

	return value
}

func assertMetricCounterValue(
	t *testing.T,
	families []*dto.MetricFamily,
	name string,
	expected float64,
) {
	t.Helper()

	for _, family := range families {
		if family.GetName() != name {
			continue
		}

		var total float64
		for _, metric := range family.GetMetric() {
			total += metric.GetCounter().GetValue()
		}

		assert.Equal(t, expected, total)
		return
	}

	require.Failf(t, "missing metric", "metric %q was not gathered", name)
}

func assertMetricGaugeValue(
	t *testing.T,
	families []*dto.MetricFamily,
	name string,
	expected float64,
) {
	t.Helper()

	for _, family := range families {
		if family.GetName() != name {
			continue
		}

		var total float64
		for _, metric := range family.GetMetric() {
			total += metric.GetGauge().GetValue()
		}

		assert.Equal(t, expected, total)
		return
	}

	require.Failf(t, "missing metric", "metric %q was not gathered", name)
}
