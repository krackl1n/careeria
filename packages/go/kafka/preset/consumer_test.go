package preset

import (
	"context"
	"testing"
	"time"

	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/krackl1n/careeria/kafka"
	consumermw "github.com/krackl1n/careeria/kafka/middleware/consumer"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsumer(t *testing.T) {
	t.Parallel()

	t.Run("panics on invalid required config", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "kafka consumer preset: consumer group is required", func() {
			Consumer(ConsumerConfig{})
		})

		require.PanicsWithValue(t, "kafka consumer preset: nil metrics", func() {
			Consumer(ConsumerConfig{ConsumerGroup: "orders-service"})
		})
	})

	t.Run("builds base preset", func(t *testing.T) {
		t.Parallel()

		middlewares := Consumer(ConsumerConfig{
			ConsumerGroup: "orders-service",
			Metrics:       newMetrics(t),
		})

		assert.Len(t, middlewares, 6)
		for _, middleware := range middlewares {
			assert.NotNil(t, middleware)
		}
	})

	t.Run("builds full preset and applies optional middlewares after base", func(t *testing.T) {
		t.Parallel()

		middlewares := Consumer(ConsumerConfig{
			ConsumerGroup: "orders-service",
			Metrics:       newMetrics(t),
			Timeout: &consumermw.TimeoutConfig{
				Provider: consumermw.StaticTimeout(time.Minute),
			},
			Validation: &consumermw.ValidationConfig{
				Validator: consumermw.EnvelopeValidatorFunc(func(kafka.ConsumedMessage) error {
					return nil
				}),
			},
		})

		assert.Len(t, middlewares, 7)

		handler := applyMiddlewares(middlewares, kafka.HandlerFunc(func(ctx context.Context, message kafka.ConsumedMessage) error {
			messageID, ok := ctxmeta.MessageID(ctx)
			assert.True(t, ok)
			assert.NotEmpty(t, messageID)
			_, ok = ctx.Deadline()
			assert.True(t, ok)
			return nil
		}))

		err := handler.Handle(context.Background(), kafka.ConsumedMessage{
			Topic:     "orders",
			Partition: 1,
			Offset:    2,
		})

		require.NoError(t, err)
	})
}

func newMetrics(t *testing.T) *consumermw.Metrics {
	t.Helper()

	metrics, err := consumermw.NewMetrics(consumermw.MetricsConfig{
		Registerer:    prometheus.NewRegistry(),
		ConsumerGroup: "orders-service",
	})
	require.NoError(t, err)

	return metrics
}

func applyMiddlewares(
	middlewares []kafka.ConsumerMiddleware,
	next kafka.Handler,
) kafka.Handler {
	for index := len(middlewares) - 1; index >= 0; index-- {
		next = middlewares[index](next)
	}

	return next
}
