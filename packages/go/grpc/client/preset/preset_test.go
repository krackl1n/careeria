package preset

import (
	"context"
	"testing"
	"time"

	"github.com/krackl1n/careeria/grpc/client/interceptor"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("panics on invalid required config", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "grpc client preset: nil metrics", func() {
			New("users", Deps{})
		})

		metrics := newMetrics(t)
		require.PanicsWithValue(t, "grpc client preset: empty peer service", func() {
			New("", Deps{Metrics: metrics})
		})
	})

	t.Run("builds base preset", func(t *testing.T) {
		t.Parallel()

		preset := New("users", Deps{Metrics: newMetrics(t)})

		assert.Len(t, preset.Unary, 3)
		assert.Len(t, preset.Stream, 3)
	})

	t.Run("builds full preset with optional interceptors", func(t *testing.T) {
		t.Parallel()

		preset := New("users", Deps{
			Metrics: newMetrics(t),
			MethodTimeout: &interceptor.MethodTimeoutDeps{
				Provider: func(string) time.Duration {
					return time.Second
				},
			},
			Authentication: &interceptor.AuthenticationDeps{
				Provider: interceptor.CredentialsProviderFunc(func(context.Context, string) (metadata.MD, error) {
					return metadata.Pairs("authorization", "bearer token"), nil
				}),
			},
		})

		assert.Len(t, preset.Unary, 5)
		assert.Len(t, preset.Stream, 5)
		for _, unary := range preset.Unary {
			assert.NotNil(t, unary)
		}
		for _, stream := range preset.Stream {
			assert.NotNil(t, stream)
		}
	})
}

func newMetrics(t *testing.T) *interceptor.Metrics {
	t.Helper()

	metrics, err := interceptor.NewMetrics(
		interceptor.MetricsConfig{},
		prometheus.NewRegistry(),
	)
	require.NoError(t, err)

	return metrics
}
