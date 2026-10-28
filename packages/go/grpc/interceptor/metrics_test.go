package interceptor

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewMetrics(t *testing.T) {
	t.Parallel()

	t.Run("registers collectors", func(t *testing.T) {
		t.Parallel()

		metrics, err := NewMetrics(MetricsConfig{
			Namespace: "toolbox",
			Subsystem: "grpc",
		}, prometheus.NewRegistry())

		require.NoError(t, err)
		require.NotNil(t, metrics)
	})

	t.Run("returns registration error", func(t *testing.T) {
		t.Parallel()

		registerer := prometheus.NewRegistry()

		_, err := NewMetrics(MetricsConfig{}, registerer)
		require.NoError(t, err)

		_, err = NewMetrics(MetricsConfig{}, registerer)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "register grpc metrics")
	})
}

func TestMetricsUnaryServerInterceptor(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil metrics", func(t *testing.T) {
		t.Parallel()

		var metrics *Metrics

		require.PanicsWithValue(t, "grpc metrics interceptor: nil metrics", func() {
			metrics.UnaryServerInterceptor()
		})
	})

	t.Run("records successful unary request", func(t *testing.T) {
		t.Parallel()

		metrics, err := NewMetrics(MetricsConfig{}, prometheus.NewRegistry())
		require.NoError(t, err)

		resp, err := metrics.UnaryServerInterceptor()(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				return "response", nil
			},
		)

		require.NoError(t, err)
		assert.Equal(t, "response", resp)
		assert.Equal(
			t,
			float64(1),
			counterValue(t, metrics.started.WithLabelValues(
				"toolbox.TestService",
				"Create",
				"unary",
			)),
		)
		assert.Equal(
			t,
			float64(1),
			counterValue(t, metrics.handled.WithLabelValues(
				"toolbox.TestService",
				"Create",
				"unary",
				codes.OK.String(),
			)),
		)
		assert.Equal(
			t,
			float64(0),
			gaugeValue(t, metrics.inFlight.WithLabelValues(
				"toolbox.TestService",
				"Create",
				"unary",
			)),
		)
	})

	t.Run("records failed unary request code", func(t *testing.T) {
		t.Parallel()

		metrics, err := NewMetrics(MetricsConfig{}, prometheus.NewRegistry())
		require.NoError(t, err)

		_, err = metrics.UnaryServerInterceptor()(
			context.Background(),
			"request",
			&grpc.UnaryServerInfo{FullMethod: testFullMethod},
			func(ctx context.Context, req any) (any, error) {
				return nil, status.Error(codes.NotFound, "not found")
			},
		)

		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Equal(
			t,
			float64(1),
			counterValue(t, metrics.handled.WithLabelValues(
				"toolbox.TestService",
				"Create",
				"unary",
				codes.NotFound.String(),
			)),
		)
	})
}

func TestMetricsStreamServerInterceptor(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil metrics", func(t *testing.T) {
		t.Parallel()

		var metrics *Metrics

		require.PanicsWithValue(t, "grpc metrics interceptor: nil metrics", func() {
			metrics.StreamServerInterceptor()
		})
	})

	t.Run("records stream request", func(t *testing.T) {
		t.Parallel()

		metrics, err := NewMetrics(MetricsConfig{}, prometheus.NewRegistry())
		require.NoError(t, err)

		err = metrics.StreamServerInterceptor()(
			nil,
			&fakeServerStream{ctx: context.Background()},
			&grpc.StreamServerInfo{
				FullMethod:     testFullMethod,
				IsClientStream: true,
				IsServerStream: true,
			},
			func(srv any, stream grpc.ServerStream) error {
				return nil
			},
		)

		require.NoError(t, err)
		assert.Equal(
			t,
			float64(1),
			counterValue(t, metrics.started.WithLabelValues(
				"toolbox.TestService",
				"Create",
				"bidi_stream",
			)),
		)
		assert.Equal(
			t,
			float64(1),
			counterValue(t, metrics.handled.WithLabelValues(
				"toolbox.TestService",
				"Create",
				"bidi_stream",
				codes.OK.String(),
			)),
		)
	})
}

func counterValue(t *testing.T, metric prometheus.Metric) float64 {
	t.Helper()

	var value dto.Metric
	require.NoError(t, metric.Write(&value))

	return value.GetCounter().GetValue()
}

func gaugeValue(t *testing.T, metric prometheus.Metric) float64 {
	t.Helper()

	var value dto.Metric
	require.NoError(t, metric.Write(&value))

	return value.GetGauge().GetValue()
}
