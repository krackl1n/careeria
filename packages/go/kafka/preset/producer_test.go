package preset

import (
	"context"
	"testing"
	"time"

	"github.com/krackl1n/careeria/kafka"
	producermw "github.com/krackl1n/careeria/kafka/middleware/producer"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProducer(t *testing.T) {
	t.Parallel()

	t.Run("panics on invalid required config", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "kafka producer preset: producer is required", func() {
			Producer(ProducerConfig{})
		})

		require.PanicsWithValue(t, "kafka producer preset: nil metrics", func() {
			Producer(ProducerConfig{Producer: "orders-producer"})
		})
	})

	t.Run("builds base preset with default validator", func(t *testing.T) {
		t.Parallel()

		middlewares := Producer(ProducerConfig{
			Producer: "orders-producer",
			Metrics:  newProducerMetrics(t),
		})

		assert.Len(t, middlewares, 5)

		publisher := applyProducerMiddlewares(middlewares, kafka.PublisherFunc(func(ctx context.Context, message kafka.Message) error {
			t.Fatal("publisher must not be called for invalid message")
			return nil
		}))

		err := publisher.Publish(context.Background(), kafka.Message{})
		require.ErrorContains(t, err, "topic is required")
	})

	t.Run("builds full preset", func(t *testing.T) {
		t.Parallel()

		middlewares := Producer(ProducerConfig{
			Producer: "orders-producer",
			Metrics:  newProducerMetrics(t),
			Timeout: &producermw.TimeoutConfig{
				Provider: producermw.StaticTimeout(time.Minute),
			},
			Validation: producermw.ValidationConfig{
				Validator: producermw.EnvelopeValidatorFunc(func(kafka.Message) error {
					return nil
				}),
			},
		})

		assert.Len(t, middlewares, 6)

		publisher := applyProducerMiddlewares(middlewares, kafka.PublisherFunc(func(ctx context.Context, message kafka.Message) error {
			_, ok := ctx.Deadline()
			assert.True(t, ok)
			return nil
		}))

		err := publisher.Publish(context.Background(), kafka.Message{
			Topic: "orders",
			Value: []byte(`{"id":"order-123"}`),
		})

		require.NoError(t, err)
	})
}

func newProducerMetrics(t *testing.T) *producermw.Metrics {
	t.Helper()

	metrics, err := producermw.NewMetrics(producermw.MetricsConfig{
		Registerer: prometheus.NewRegistry(),
		Producer:   "orders-producer",
	})
	require.NoError(t, err)

	return metrics
}

func applyProducerMiddlewares(
	middlewares []kafka.ProducerMiddleware,
	next kafka.Publisher,
) kafka.Publisher {
	for index := len(middlewares) - 1; index >= 0; index-- {
		next = middlewares[index](next)
	}

	return next
}
