package consumer

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
	"go.opentelemetry.io/otel/trace"
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

	t.Run("panics on nil handler", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "kafka consumer correlation middleware: nil handler", func() {
			Correlation(CorrelationConfig{})(nil)
		})
	})

	t.Run("extracts identifiers from headers", func(t *testing.T) {
		t.Parallel()

		message := testMessage()
		message.Headers = []kafka.Header{
			{Key: kafka.HeaderMessageID, Value: []byte("msg-123")},
			{Key: kafka.HeaderCorrelationID, Value: []byte("corr-123")},
			{Key: kafka.HeaderCausationID, Value: []byte("cause-123")},
		}

		err := Correlation(CorrelationConfig{})(kafka.HandlerFunc(func(ctx context.Context, message kafka.ConsumedMessage) error {
			assert.Equal(t, "msg-123", contextMetadata(t, ctx, ctxmeta.MessageID))
			assert.Equal(t, "corr-123", contextMetadata(t, ctx, ctxmeta.CorrelationID))
			assert.Equal(t, "cause-123", contextMetadata(t, ctx, ctxmeta.CausationID))
			return nil
		})).Handle(context.Background(), message)

		require.NoError(t, err)
	})

	t.Run("uses fallback message id and correlation id", func(t *testing.T) {
		t.Parallel()

		message := testMessage()

		err := Correlation(CorrelationConfig{})(kafka.HandlerFunc(func(ctx context.Context, message kafka.ConsumedMessage) error {
			assert.Equal(t, "orders:2:42", contextMetadata(t, ctx, ctxmeta.MessageID))
			assert.Equal(t, "orders:2:42", contextMetadata(t, ctx, ctxmeta.CorrelationID))
			assert.Empty(t, contextMetadata(t, ctx, ctxmeta.CausationID))
			return nil
		})).Handle(context.Background(), message)

		require.NoError(t, err)
	})
}

func TestTimeout(t *testing.T) {
	t.Parallel()

	t.Run("panics on invalid config", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "kafka consumer timeout middleware: nil provider", func() {
			Timeout(TimeoutConfig{})
		})
		require.PanicsWithValue(t, "kafka consumer timeout middleware: nil handler", func() {
			Timeout(TimeoutConfig{Provider: StaticTimeout(time.Second)})(nil)
		})
	})

	t.Run("skips non-positive timeout", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		err := Timeout(TimeoutConfig{Provider: StaticTimeout(0)})(kafka.HandlerFunc(func(actualCtx context.Context, message kafka.ConsumedMessage) error {
			assert.True(t, ctx == actualCtx)
			_, ok := actualCtx.Deadline()
			assert.False(t, ok)
			return nil
		})).Handle(ctx, testMessage())

		require.NoError(t, err)
	})

	t.Run("applies timeout", func(t *testing.T) {
		t.Parallel()

		err := Timeout(TimeoutConfig{Provider: StaticTimeout(time.Minute)})(kafka.HandlerFunc(func(ctx context.Context, message kafka.ConsumedMessage) error {
			deadline, ok := ctx.Deadline()
			require.True(t, ok)
			assert.WithinDuration(t, time.Now().Add(time.Minute), deadline, 2*time.Second)
			return nil
		})).Handle(context.Background(), testMessage())

		require.NoError(t, err)
	})

	t.Run("returns handler error", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("handler failed")

		err := Timeout(TimeoutConfig{Provider: StaticTimeout(time.Minute)})(kafka.HandlerFunc(func(context.Context, kafka.ConsumedMessage) error {
			return expectedErr
		})).Handle(context.Background(), testMessage())

		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("returns deadline exceeded when handler ignores expired context", func(t *testing.T) {
		t.Parallel()

		err := Timeout(TimeoutConfig{Provider: StaticTimeout(time.Nanosecond)})(kafka.HandlerFunc(func(context.Context, kafka.ConsumedMessage) error {
			time.Sleep(time.Millisecond)
			return nil
		})).Handle(context.Background(), testMessage())

		require.ErrorIs(t, err, context.DeadlineExceeded)
	})
}

func TestValidation(t *testing.T) {
	t.Parallel()

	t.Run("panics on invalid config", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "kafka consumer validation middleware: nil validator", func() {
			Validation(ValidationConfig{})
		})
		require.PanicsWithValue(t, "kafka consumer validation middleware: nil handler", func() {
			Validation(ValidationConfig{Validator: EnvelopeValidatorFunc(func(kafka.ConsumedMessage) error {
				return nil
			})})(nil)
		})
	})

	t.Run("returns validation error without calling next", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("invalid envelope")

		err := Validation(ValidationConfig{
			Validator: EnvelopeValidatorFunc(func(kafka.ConsumedMessage) error {
				return expectedErr
			}),
		})(kafka.HandlerFunc(func(context.Context, kafka.ConsumedMessage) error {
			t.Fatal("next must not be called")
			return nil
		})).Handle(context.Background(), testMessage())

		require.ErrorIs(t, err, expectedErr)
		assert.Contains(t, err.Error(), "validate kafka message")
	})

	t.Run("calls next when message is valid", func(t *testing.T) {
		t.Parallel()

		called := false

		err := Validation(ValidationConfig{
			Validator: EnvelopeValidatorFunc(func(kafka.ConsumedMessage) error {
				return nil
			}),
		})(kafka.HandlerFunc(func(context.Context, kafka.ConsumedMessage) error {
			called = true
			return nil
		})).Handle(context.Background(), testMessage())

		require.NoError(t, err)
		assert.True(t, called)
	})
}

func TestLogging(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil handler", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "kafka consumer logging middleware: nil handler", func() {
			Logging(LoggingConfig{})(nil)
		})
	})

	tests := []struct {
		name       string
		err        error
		panicValue any
		wantLevel  string
		wantResult string
	}{
		{
			name:       "success logs info",
			wantLevel:  "INFO",
			wantResult: resultSuccess,
		},
		{
			name:       "handler error logs error",
			err:        errors.New("handler failed"),
			wantLevel:  "ERROR",
			wantResult: resultError,
		},
		{
			name:       "canceled logs warn",
			err:        context.Canceled,
			wantLevel:  "WARN",
			wantResult: resultCanceled,
		},
		{
			name:       "deadline exceeded logs warn",
			err:        context.DeadlineExceeded,
			wantLevel:  "WARN",
			wantResult: resultTimeout,
		},
		{
			name:       "panic logs error and repanics",
			panicValue: "boom",
			wantLevel:  "ERROR",
			wantResult: resultPanic,
		},
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

			handler := Logging(LoggingConfig{
				Logger:        logger,
				ConsumerGroup: "orders-service",
			})(kafka.HandlerFunc(func(context.Context, kafka.ConsumedMessage) error {
				if tt.panicValue != nil {
					panic(tt.panicValue)
				}

				return tt.err
			}))

			if tt.panicValue != nil {
				require.PanicsWithValue(t, tt.panicValue, func() {
					_ = handler.Handle(ctx, testMessage())
				})
			} else {
				err := handler.Handle(ctx, testMessage())
				if tt.err != nil {
					require.ErrorIs(t, err, tt.err)
				} else {
					require.NoError(t, err)
				}
			}

			assert.Contains(t, logs.String(), tt.wantLevel)
			assert.Contains(t, logs.String(), tt.wantResult)
			assert.Contains(t, logs.String(), "orders-service")
			assert.Contains(t, logs.String(), "orders")
			assert.Contains(t, logs.String(), "msg-123")
			assert.Contains(t, logs.String(), "corr-123")
			assert.Contains(t, logs.String(), "cause-123")
		})
	}
}

func TestRecovery(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil handler", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "kafka consumer recovery middleware: nil handler", func() {
			Recovery(nil, nil)(nil)
		})
	})

	t.Run("passes through handler error", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("handler failed")

		err := Recovery(nil, nil)(kafka.HandlerFunc(func(context.Context, kafka.ConsumedMessage) error {
			return expectedErr
		})).Handle(context.Background(), testMessage())

		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("recovers panic into error", func(t *testing.T) {
		t.Parallel()

		var logs bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&logs, nil))

		err := Recovery(logger, nil)(kafka.HandlerFunc(func(context.Context, kafka.ConsumedMessage) error {
			panic("boom")
		})).Handle(context.Background(), testMessage())

		require.ErrorIs(t, err, ErrHandlerPanic)
		assert.Contains(t, err.Error(), "boom")
		assert.Contains(t, logs.String(), "kafka consumer panic recovered")
		assert.Contains(t, logs.String(), "boom")
	})

	t.Run("uses panic handler error", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("send to dlq failed")
		panicHandler := panicHandlerFunc(func(ctx context.Context, message kafka.ConsumedMessage, recovered any) error {
			assert.Equal(t, "boom", recovered)
			return expectedErr
		})

		err := Recovery(nil, panicHandler)(kafka.HandlerFunc(func(context.Context, kafka.ConsumedMessage) error {
			panic("boom")
		})).Handle(context.Background(), testMessage())

		require.ErrorIs(t, err, expectedErr)
	})
}

func TestMetrics(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil metrics or handler", func(t *testing.T) {
		t.Parallel()

		var metrics *Metrics
		require.PanicsWithValue(t, "kafka consumer metrics middleware: nil metrics", func() {
			metrics.Middleware()
		})

		metrics, err := NewMetrics(MetricsConfig{
			Registerer:    prometheus.NewRegistry(),
			ConsumerGroup: "orders-service",
		})
		require.NoError(t, err)

		require.PanicsWithValue(t, "kafka consumer metrics middleware: nil handler", func() {
			metrics.Middleware()(nil)
		})
	})

	t.Run("requires consumer group", func(t *testing.T) {
		t.Parallel()

		metrics, err := NewMetrics(MetricsConfig{})
		require.Nil(t, metrics)
		require.EqualError(t, err, "kafka consumer metrics: consumer group is required")
	})

	t.Run("reuses registered collectors", func(t *testing.T) {
		t.Parallel()

		registerer := prometheus.NewRegistry()
		first, err := NewMetrics(MetricsConfig{
			Registerer:    registerer,
			ConsumerGroup: "orders-service",
		})
		require.NoError(t, err)

		second, err := NewMetrics(MetricsConfig{
			Registerer:    registerer,
			ConsumerGroup: "orders-service",
		})
		require.NoError(t, err)

		assert.Same(t, first.started, second.started)
		assert.Same(t, first.handled, second.handled)
		assert.Same(t, first.duration, second.duration)
		assert.Same(t, first.inFlight, second.inFlight)
	})

	t.Run("records success and error results", func(t *testing.T) {
		t.Parallel()

		registerer := prometheus.NewRegistry()
		metrics, err := NewMetrics(MetricsConfig{
			Registerer:    registerer,
			ConsumerGroup: "orders-service",
		})
		require.NoError(t, err)

		handler := metrics.Middleware()(kafka.HandlerFunc(func(ctx context.Context, message kafka.ConsumedMessage) error {
			if string(message.Key) == "error" {
				return io.ErrUnexpectedEOF
			}

			return nil
		}))

		require.NoError(t, handler.Handle(context.Background(), testMessage()))

		failed := testMessage()
		failed.Key = []byte("error")
		require.ErrorIs(t, handler.Handle(context.Background(), failed), io.ErrUnexpectedEOF)

		families, err := registerer.Gather()
		require.NoError(t, err)

		assertMetricCounterValue(t, families, "messages_started_total", 2)
		assertMetricCounterValue(t, families, "messages_handled_total", 2)
		assertMetricGaugeValue(t, families, "messages_in_flight", 0)
	})

	t.Run("decrements in flight on panic", func(t *testing.T) {
		t.Parallel()

		registerer := prometheus.NewRegistry()
		metrics, err := NewMetrics(MetricsConfig{
			Registerer:    registerer,
			ConsumerGroup: "orders-service",
		})
		require.NoError(t, err)

		handler := metrics.Middleware()(kafka.HandlerFunc(func(context.Context, kafka.ConsumedMessage) error {
			panic("boom")
		}))

		require.PanicsWithValue(t, "boom", func() {
			_ = handler.Handle(context.Background(), testMessage())
		})

		families, err := registerer.Gather()
		require.NoError(t, err)

		assertMetricCounterValue(t, families, "messages_started_total", 1)
		assertMetricCounterValue(t, families, "messages_handled_total", 1)
		assertMetricGaugeValue(t, families, "messages_in_flight", 0)
	})
}

func TestTracing(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil handler", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "kafka consumer tracing middleware: nil handler", func() {
			Tracing(TracingConfig{})(nil)
		})
	})

	t.Run("creates consumer span and extracts parent context", func(t *testing.T) {
		t.Parallel()

		recorder := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
		t.Cleanup(func() {
			require.NoError(t, provider.Shutdown(context.Background()))
		})

		message := testMessage()
		message.Headers = []kafka.Header{
			{
				Key:   "traceparent",
				Value: []byte("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"),
			},
		}

		err := Tracing(TracingConfig{
			Tracer:     provider.Tracer("test"),
			Propagator: propagation.TraceContext{},
		})(kafka.HandlerFunc(func(ctx context.Context, message kafka.ConsumedMessage) error {
			assert.True(t, trace.SpanFromContext(ctx).SpanContext().IsValid())
			return nil
		})).Handle(context.Background(), message)

		require.NoError(t, err)

		spans := recorder.Ended()
		require.Len(t, spans, 1)
		assert.Equal(t, "process orders", spans[0].Name())
		assert.Equal(t, codes.Unset, spans[0].Status().Code)
		assert.True(t, spans[0].Parent().IsValid())
	})

	t.Run("records handler error", func(t *testing.T) {
		t.Parallel()

		recorder := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
		t.Cleanup(func() {
			require.NoError(t, provider.Shutdown(context.Background()))
		})

		expectedErr := errors.New("handler failed")

		err := Tracing(TracingConfig{
			Tracer: provider.Tracer("test"),
		})(kafka.HandlerFunc(func(context.Context, kafka.ConsumedMessage) error {
			return expectedErr
		})).Handle(context.Background(), testMessage())

		require.ErrorIs(t, err, expectedErr)

		spans := recorder.Ended()
		require.Len(t, spans, 1)
		assert.Equal(t, codes.Error, spans[0].Status().Code)
		assert.Contains(t, spans[0].Status().Description, expectedErr.Error())
	})

	t.Run("records panic and repanics", func(t *testing.T) {
		t.Parallel()

		recorder := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
		t.Cleanup(func() {
			require.NoError(t, provider.Shutdown(context.Background()))
		})

		handler := Tracing(TracingConfig{
			Tracer: provider.Tracer("test"),
		})(kafka.HandlerFunc(func(context.Context, kafka.ConsumedMessage) error {
			panic("boom")
		}))

		require.PanicsWithValue(t, "boom", func() {
			_ = handler.Handle(context.Background(), testMessage())
		})

		spans := recorder.Ended()
		require.Len(t, spans, 1)
		assert.Equal(t, codes.Error, spans[0].Status().Code)
		assert.Equal(t, "panic", spans[0].Status().Description)
	})
}

func TestHeaderCarrier(t *testing.T) {
	t.Parallel()

	carrier := newHeaderCarrier([]kafka.Header{
		{Key: "TraceParent", Value: []byte("trace")},
	})

	assert.Equal(t, "trace", carrier.Get("traceparent"))

	carrier.Set("Baggage", "value")
	assert.Equal(t, "value", carrier.Get("baggage"))
	assert.ElementsMatch(t, []string{"traceparent", "baggage"}, carrier.Keys())
}

type panicHandlerFunc func(context.Context, kafka.ConsumedMessage, any) error

func (fn panicHandlerFunc) Handle(
	ctx context.Context,
	message kafka.ConsumedMessage,
	recovered any,
) error {
	return fn(ctx, message, recovered)
}

func testMessage() kafka.ConsumedMessage {
	return kafka.ConsumedMessage{
		Topic:     "orders",
		Partition: 2,
		Offset:    42,
		Key:       []byte("order-key"),
		Value:     []byte(`{"id":"order-123"}`),
	}
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
