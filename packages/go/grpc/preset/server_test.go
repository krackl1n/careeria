package preset

import (
	"context"
	"testing"
	"time"

	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/krackl1n/careeria/grpc/interceptor"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type authenticatorFunc func(context.Context, string) (ctxmeta.Principal, error)

func (fn authenticatorFunc) Authenticate(ctx context.Context, fullMethod string) (ctxmeta.Principal, error) {
	return fn(ctx, fullMethod)
}

type authorizerFunc func(context.Context, string) error

func (fn authorizerFunc) Authorize(ctx context.Context, fullMethod string) error {
	return fn(ctx, fullMethod)
}

type timeoutProviderFunc func(string) time.Duration

func (fn timeoutProviderFunc) Timeout(fullMethod string) time.Duration {
	return fn(fullMethod)
}

type idempotencyManagerFunc func(
	context.Context,
	string,
	string,
	any,
	interceptor.IdempotencyOperation,
) (any, error)

func (fn idempotencyManagerFunc) Execute(
	ctx context.Context,
	scope string,
	key string,
	request any,
	operation interceptor.IdempotencyOperation,
) (any, error) {
	return fn(ctx, scope, key, request, operation)
}

func TestServer(t *testing.T) {
	t.Parallel()

	t.Run("panics on nil metrics", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "grpc server preset: nil metrics", func() {
			Server(ServerConfig{})
		})
	})

	t.Run("builds base server preset", func(t *testing.T) {
		t.Parallel()

		preset := Server(ServerConfig{
			Metrics: newTestMetrics(t),
		})

		require.NotNil(t, preset.StatsHandler)
		assert.Len(t, preset.Unary, 6)
		assert.Len(t, preset.Stream, 6)
	})

	t.Run("builds server preset with optional interceptors", func(t *testing.T) {
		t.Parallel()

		preset := Server(ServerConfig{
			Metrics: newTestMetrics(t),
			Authentication: &interceptor.AuthenticationConfig{
				Authenticator: authenticatorFunc(func(context.Context, string) (ctxmeta.Principal, error) {
					return ctxmeta.Principal{SubjectID: "principal"}, nil
				}),
			},
			TenantContext: &interceptor.TenantContextConfig{},
			Authorization: &interceptor.AuthorizationConfig{
				Authorizer: authorizerFunc(func(context.Context, string) error {
					return nil
				}),
			},
			MethodTimeout: &interceptor.MethodTimeoutConfig{
				Provider: timeoutProviderFunc(func(string) time.Duration {
					return time.Second
				}),
			},
			Idempotency: &interceptor.IdempotencyConfig{
				Manager: idempotencyManagerFunc(func(
					ctx context.Context,
					scope string,
					key string,
					request any,
					operation interceptor.IdempotencyOperation,
				) (any, error) {
					return operation(ctx)
				}),
			},
		})

		require.NotNil(t, preset.StatsHandler)
		assert.Len(t, preset.Unary, 11)
		assert.Len(t, preset.Stream, 10)
	})
}

func newTestMetrics(t *testing.T) *interceptor.Metrics {
	t.Helper()

	metrics, err := interceptor.NewMetrics(interceptor.MetricsConfig{}, prometheus.NewRegistry())
	require.NoError(t, err)

	return metrics
}
