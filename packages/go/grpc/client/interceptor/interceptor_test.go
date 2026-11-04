package interceptor

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/krackl1n/careeria/grpc/matcher"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUnaryAuthentication(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil provider", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "grpc client authentication interceptor: nil credentials provider", func() {
			UnaryAuthentication(AuthenticationDeps{})
		})
	})

	t.Run("skips unmatched method", func(t *testing.T) {
		t.Parallel()

		called := false
		err := UnaryAuthentication(AuthenticationDeps{
			Provider: CredentialsProviderFunc(func(context.Context, string) (metadata.MD, error) {
				called = true
				return metadata.Pairs("authorization", "bearer token"), nil
			}),
			Matcher: matcher.MatchMethods("Other"),
		})(context.Background(), "/toolbox.Service/Create", nil, nil, nil, func(
			ctx context.Context,
			method string,
			req any,
			reply any,
			cc *grpc.ClientConn,
			opts ...grpc.CallOption,
		) error {
			assert.Empty(t, metadataValue(ctx, "authorization"))
			return nil
		})

		require.NoError(t, err)
		assert.False(t, called)
	})

	t.Run("adds outgoing metadata", func(t *testing.T) {
		t.Parallel()

		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "old"))

		err := UnaryAuthentication(AuthenticationDeps{
			Provider: CredentialsProviderFunc(func(context.Context, string) (metadata.MD, error) {
				return metadata.Pairs("Authorization", "new", "x-api-key", "key"), nil
			}),
		})(ctx, "/toolbox.Service/Create", nil, nil, nil, func(
			ctx context.Context,
			method string,
			req any,
			reply any,
			cc *grpc.ClientConn,
			opts ...grpc.CallOption,
		) error {
			assert.Equal(t, "new", metadataValue(ctx, "authorization"))
			assert.Equal(t, "key", metadataValue(ctx, "x-api-key"))
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, "old", metadataValue(ctx, "authorization"))
	})

	t.Run("returns provider error", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("credentials failed")

		err := UnaryAuthentication(AuthenticationDeps{
			Provider: CredentialsProviderFunc(func(context.Context, string) (metadata.MD, error) {
				return nil, expectedErr
			}),
		})(context.Background(), "/toolbox.Service/Create", nil, nil, nil, func(
			context.Context,
			string,
			any,
			any,
			*grpc.ClientConn,
			...grpc.CallOption,
		) error {
			t.Fatal("invoker must not be called")
			return nil
		})

		require.ErrorIs(t, err, expectedErr)
	})
}

func TestStreamAuthentication(t *testing.T) {
	t.Parallel()

	stream := &testClientStream{}

	actualStream, err := StreamAuthentication(AuthenticationDeps{
		Provider: CredentialsProviderFunc(func(context.Context, string) (metadata.MD, error) {
			return metadata.Pairs("authorization", "bearer token"), nil
		}),
	})(context.Background(), &grpc.StreamDesc{ServerStreams: true}, nil, "/toolbox.Service/Watch", func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		assert.Equal(t, "bearer token", metadataValue(ctx, "authorization"))
		return stream, nil
	})

	require.NoError(t, err)
	assert.Same(t, stream, actualStream)
}

func TestRequestID(t *testing.T) {
	t.Parallel()

	t.Run("adds request id when missing", func(t *testing.T) {
		t.Parallel()

		ctx := ctxmeta.WithRequestID(context.Background(), "req-123")
		ctx = ctxmeta.WithCorrelationID(ctx, "corr-123")
		err := UnaryRequestID()(ctx, "/toolbox.Service/Create", nil, nil, nil, func(
			ctx context.Context,
			method string,
			req any,
			reply any,
			cc *grpc.ClientConn,
			opts ...grpc.CallOption,
		) error {
			assert.Equal(t, "req-123", metadataValue(ctx, RequestIDMetadataKey))
			assert.Equal(t, "corr-123", metadataValue(ctx, CorrelationIDMetadataKey))
			return nil
		})

		require.NoError(t, err)
	})

	t.Run("preserves existing request id and skips empty value", func(t *testing.T) {
		t.Parallel()

		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(RequestIDMetadataKey, "existing"))

		err := UnaryRequestID()(ctx, "/toolbox.Service/Create", nil, nil, nil, func(
			ctx context.Context,
			method string,
			req any,
			reply any,
			cc *grpc.ClientConn,
			opts ...grpc.CallOption,
		) error {
			assert.Equal(t, "existing", metadataValue(ctx, RequestIDMetadataKey))
			return nil
		})
		require.NoError(t, err)

		err = UnaryRequestID()(context.Background(), "/toolbox.Service/Create", nil, nil, nil, func(
			ctx context.Context,
			method string,
			req any,
			reply any,
			cc *grpc.ClientConn,
			opts ...grpc.CallOption,
		) error {
			assert.Empty(t, metadataValue(ctx, RequestIDMetadataKey))
			return nil
		})
		require.NoError(t, err)
	})
}

func TestMethodTimeout(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil provider", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "grpc client method timeout interceptor: nil timeout provider", func() {
			UnaryMethodTimeout(MethodTimeoutDeps{})
		})
		require.PanicsWithValue(t, "grpc client method timeout interceptor: nil timeout provider", func() {
			StreamMethodTimeout(MethodTimeoutDeps{})
		})
	})

	t.Run("skips unmatched method", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		err := UnaryMethodTimeout(MethodTimeoutDeps{
			Provider: func(string) time.Duration {
				return time.Second
			},
			Matcher: matcher.MatchMethods("Other"),
		})(ctx, "/toolbox.Service/Create", nil, nil, nil, func(
			actualCtx context.Context,
			method string,
			req any,
			reply any,
			cc *grpc.ClientConn,
			opts ...grpc.CallOption,
		) error {
			assert.True(t, ctx == actualCtx)
			_, ok := actualCtx.Deadline()
			assert.False(t, ok)
			return nil
		})

		require.NoError(t, err)
	})

	t.Run("applies deadline and preserves shorter existing deadline", func(t *testing.T) {
		t.Parallel()

		err := UnaryMethodTimeout(MethodTimeoutDeps{
			Provider: func(string) time.Duration {
				return time.Minute
			},
		})(context.Background(), "/toolbox.Service/Create", nil, nil, nil, func(
			ctx context.Context,
			method string,
			req any,
			reply any,
			cc *grpc.ClientConn,
			opts ...grpc.CallOption,
		) error {
			deadline, ok := ctx.Deadline()
			require.True(t, ok)
			assert.WithinDuration(t, time.Now().Add(time.Minute), deadline, 2*time.Second)
			return nil
		})
		require.NoError(t, err)

		shortCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()

		err = UnaryMethodTimeout(MethodTimeoutDeps{
			Provider: func(string) time.Duration {
				return time.Minute
			},
		})(shortCtx, "/toolbox.Service/Create", nil, nil, nil, func(
			ctx context.Context,
			method string,
			req any,
			reply any,
			cc *grpc.ClientConn,
			opts ...grpc.CallOption,
		) error {
			assert.True(t, shortCtx == ctx)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("cancels stream timeout when stream completes", func(t *testing.T) {
		t.Parallel()

		var streamCtx context.Context
		stream, err := StreamMethodTimeout(MethodTimeoutDeps{
			Provider: func(string) time.Duration {
				return time.Minute
			},
		})(context.Background(), &grpc.StreamDesc{ServerStreams: true}, nil, "/toolbox.Service/Watch", func(
			ctx context.Context,
			desc *grpc.StreamDesc,
			cc *grpc.ClientConn,
			method string,
			opts ...grpc.CallOption,
		) (grpc.ClientStream, error) {
			streamCtx = ctx
			return &testClientStream{recvErr: io.EOF}, nil
		})

		require.NoError(t, err)
		require.NotNil(t, streamCtx)

		err = stream.RecvMsg(new(string))
		require.ErrorIs(t, err, io.EOF)
		assert.Eventually(t, func() bool {
			return errors.Is(streamCtx.Err(), context.Canceled)
		}, time.Second, time.Millisecond)
	})
}

func TestValidation(t *testing.T) {
	t.Parallel()

	t.Run("unary validates ValidateAll before Validate", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("invalid all")

		err := UnaryValidation()(context.Background(), "/toolbox.Service/Create", validationMessage{
			validateAllErr: expectedErr,
			validateErr:    errors.New("must not be used"),
		}, nil, nil, func(
			context.Context,
			string,
			any,
			any,
			*grpc.ClientConn,
			...grpc.CallOption,
		) error {
			t.Fatal("invoker must not be called")
			return nil
		})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Equal(t, "request validation failed", status.Convert(err).Message())
	})

	t.Run("stream validates SendMsg", func(t *testing.T) {
		t.Parallel()

		stream, err := StreamValidation()(context.Background(), &grpc.StreamDesc{ClientStreams: true}, nil, "/toolbox.Service/Create", func(
			context.Context,
			*grpc.StreamDesc,
			*grpc.ClientConn,
			string,
			...grpc.CallOption,
		) (grpc.ClientStream, error) {
			return &testClientStream{}, nil
		})
		require.NoError(t, err)

		err = stream.SendMsg(validateOnlyMessage{validateErr: errors.New("invalid")})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}

func TestLogging(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		wantLevel string
		wantCode  string
	}{
		{
			name:      "success logs info",
			wantLevel: "INFO",
			wantCode:  "OK",
		},
		{
			name:      "invalid argument logs warn",
			err:       status.Error(codes.InvalidArgument, "invalid"),
			wantLevel: "WARN",
			wantCode:  "InvalidArgument",
		},
		{
			name:      "unknown logs error",
			err:       errors.New("boom"),
			wantLevel: "ERROR",
			wantCode:  "Unknown",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, nil))
			cc := newClientConn(t)
			ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(RequestIDMetadataKey, "req-123"))

			err := UnaryLogging(logger)(ctx, "/toolbox.Service/Create", nil, nil, cc, func(
				context.Context,
				string,
				any,
				any,
				*grpc.ClientConn,
				...grpc.CallOption,
			) error {
				return tt.err
			})

			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}

			assert.Contains(t, logs.String(), tt.wantLevel)
			assert.Contains(t, logs.String(), "grpc client request completed")
			assert.Contains(t, logs.String(), tt.wantCode)
			assert.Contains(t, logs.String(), "toolbox.Service")
			assert.Contains(t, logs.String(), "Create")
			assert.Contains(t, logs.String(), "req-123")
		})
	}
}

func TestMetrics(t *testing.T) {
	t.Parallel()

	t.Run("reuses collectors on duplicate registration", func(t *testing.T) {
		t.Parallel()

		registerer := prometheus.NewRegistry()
		metrics, err := NewMetrics(MetricsConfig{}, registerer)
		require.NoError(t, err)
		require.NotNil(t, metrics)

		second, err := NewMetrics(MetricsConfig{}, registerer)
		require.NoError(t, err)
		require.Same(t, metrics.started, second.started)
		require.Same(t, metrics.handled, second.handled)
		require.Same(t, metrics.duration, second.duration)
		require.Same(t, metrics.inFlight, second.inFlight)
	})

	t.Run("panics on nil metrics", func(t *testing.T) {
		t.Parallel()

		var metrics *Metrics
		require.PanicsWithValue(t, "grpc client metrics interceptor: nil metrics", func() {
			metrics.UnaryClientInterceptor("users")
		})
		require.PanicsWithValue(t, "grpc client metrics interceptor: nil metrics", func() {
			metrics.StreamClientInterceptor("users")
		})
	})

	t.Run("records unary request", func(t *testing.T) {
		t.Parallel()

		registerer := prometheus.NewRegistry()
		metrics, err := NewMetrics(MetricsConfig{}, registerer)
		require.NoError(t, err)

		err = metrics.UnaryClientInterceptor(" users ")(context.Background(), "/toolbox.Service/Create", nil, nil, nil, func(
			context.Context,
			string,
			any,
			any,
			*grpc.ClientConn,
			...grpc.CallOption,
		) error {
			return status.Error(codes.Unavailable, "down")
		})
		require.Error(t, err)

		families, err := registerer.Gather()
		require.NoError(t, err)

		assertMetricCounterValue(t, families, "grpc_client_requests_started_total", 1)
		assertMetricCounterValue(t, families, "grpc_client_requests_handled_total", 1)
		assertMetricGaugeValue(t, families, "grpc_client_requests_in_flight", 0)
	})

	t.Run("records stream completion", func(t *testing.T) {
		t.Parallel()

		registerer := prometheus.NewRegistry()
		metrics, err := NewMetrics(MetricsConfig{}, registerer)
		require.NoError(t, err)

		stream, err := metrics.StreamClientInterceptor("")(
			context.Background(),
			&grpc.StreamDesc{ServerStreams: true},
			nil,
			"/toolbox.Service/Watch",
			func(context.Context, *grpc.StreamDesc, *grpc.ClientConn, string, ...grpc.CallOption) (grpc.ClientStream, error) {
				return &testClientStream{recvErr: io.EOF}, nil
			},
		)
		require.NoError(t, err)

		err = stream.RecvMsg(new(string))
		require.ErrorIs(t, err, io.EOF)

		families, err := registerer.Gather()
		require.NoError(t, err)

		assertMetricCounterValue(t, families, "grpc_client_requests_started_total", 1)
		assertMetricCounterValue(t, families, "grpc_client_requests_handled_total", 1)
		assertMetricGaugeValue(t, families, "grpc_client_requests_in_flight", 0)
	})
}

func TestFinishingClientStream(t *testing.T) {
	t.Parallel()

	t.Run("finishes once on first terminal error", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("header failed")
		calls := 0
		stream := wrapClientStream(
			&testClientStream{
				headerErr: expectedErr,
				sendErr:   errors.New("send failed"),
			},
			context.Background(),
			&grpc.StreamDesc{ServerStreams: true},
			func(err error) {
				calls++
				assert.ErrorIs(t, err, expectedErr)
			},
		)

		_, err := stream.Header()
		require.ErrorIs(t, err, expectedErr)

		err = stream.SendMsg("message")
		require.Error(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("ignores send EOF and finishes client stream after successful recv", func(t *testing.T) {
		t.Parallel()

		calls := 0
		stream := wrapClientStream(
			&testClientStream{sendErr: io.EOF},
			context.Background(),
			&grpc.StreamDesc{ClientStreams: true},
			func(err error) {
				calls++
				assert.NoError(t, err)
			},
		)

		err := stream.SendMsg("message")
		require.ErrorIs(t, err, io.EOF)
		assert.Equal(t, 0, calls)

		err = stream.RecvMsg(new(string))
		require.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("finishes on watched context cancellation", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		finished := make(chan error, 1)

		_ = wrapClientStream(
			&testClientStream{},
			ctx,
			&grpc.StreamDesc{ServerStreams: true},
			func(err error) {
				finished <- err
			},
		)

		cancel()

		require.Eventually(t, func() bool {
			select {
			case err := <-finished:
				return errors.Is(err, context.Canceled)
			default:
				return false
			}
		}, time.Second, time.Millisecond)
	})
}

func TestMetadataHelpers(t *testing.T) {
	t.Parallel()

	t.Run("normalizes keys and respects overwrite flag", func(t *testing.T) {
		t.Parallel()

		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("X-Test", "old"))
		sameCtx := withMetadata(ctx, metadata.Pairs(" x-test ", "new"), false)
		assert.True(t, ctx == sameCtx)
		assert.Equal(t, "old", metadataValue(sameCtx, "x-test"))

		overwrittenCtx := withMetadata(ctx, metadata.Pairs(" x-test ", "new"), true)
		assert.False(t, ctx == overwrittenCtx)
		assert.Equal(t, "new", metadataValue(overwrittenCtx, "X-Test"))
	})

	t.Run("ignores empty values", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		assert.True(t, ctx == withMetadataValue(ctx, "", "value", false))
		assert.True(t, ctx == withMetadataValue(ctx, "x-test", "", false))
		assert.True(t, ctx == withMetadata(ctx, metadata.MD{" ": []string{"value"}}, true))
		assert.Empty(t, metadataValue(ctx, "x-test"))
	})
}

type validationMessage struct {
	validateAllErr error
	validateErr    error
}

func (m validationMessage) ValidateAll() error {
	return m.validateAllErr
}

func (m validationMessage) Validate() error {
	return m.validateErr
}

type validateOnlyMessage struct {
	validateErr error
}

func (m validateOnlyMessage) Validate() error {
	return m.validateErr
}

type testClientStream struct {
	grpc.ClientStream

	headerErr error
	sendErr   error
	recvErr   error
	closeErr  error
}

func (s *testClientStream) Header() (metadata.MD, error) {
	return metadata.MD{}, s.headerErr
}

func (s *testClientStream) SendMsg(any) error {
	return s.sendErr
}

func (s *testClientStream) RecvMsg(any) error {
	return s.recvErr
}

func (s *testClientStream) CloseSend() error {
	return s.closeErr
}

func newClientConn(t *testing.T) *grpc.ClientConn {
	t.Helper()

	conn, err := grpc.NewClient("dns:///example.test", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})

	return conn
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
