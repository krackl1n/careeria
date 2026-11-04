package preset

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/krackl1n/careeria/ctxmeta"
	httphandler "github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/middleware"

	"github.com/stretchr/testify/require"
)

type testAuthenticator struct{}

func (testAuthenticator) Authenticate(*http.Request) (ctxmeta.Principal, error) {
	return ctxmeta.Principal{SubjectID: "user-1"}, nil
}

type testInternalAuthenticator struct{}

func (testInternalAuthenticator) AuthenticateInternal(r *http.Request) error {
	return nil
}

type testAuthorizer struct{}

func (testAuthorizer) Authorize(r *http.Request) error {
	return nil
}

type testVerifier struct{}

func (testVerifier) Verify(r *http.Request, body []byte) error {
	return nil
}

type testTenantProvider struct{}

func (testTenantProvider) ResolveTenant(r *http.Request) (string, error) {
	return "tenant-1", nil
}

func TestPresetsComposeExpectedMiddlewares(t *testing.T) {
	t.Parallel()

	errorHandler := httphandler.NewErrorHandler(nil)

	tests := []struct {
		name string
		got  []func(http.Handler) http.Handler
		want int
	}{
		{
			name: "base",
			got:  Base(BaseConfig{}, BaseDeps{ErrorHandler: errorHandler}),
			want: 5,
		},
		{
			name: "debug",
			got:  Debug(DebugConfig{ErrorHandler: errorHandler, Authenticator: testInternalAuthenticator{}}),
			want: 3,
		},
		{
			name: "gateway without rate limit",
			got:  Gateway(GatewayConfig{Base: BaseConfig{}}, GatewayDeps{ErrorHandler: errorHandler}),
			want: 11,
		},
		{
			name: "health",
			got:  Health(nil, errorHandler),
			want: 2,
		},
		{
			name: "idempotent",
			got: Idempotent(IdempotentConfig{
				ErrorHandler: errorHandler,
				Idempotency: middleware.IdempotencyConfig{
					Store: &testIdempotencyStore{},
				},
			}),
			want: 1,
		},
		{
			name: "internal",
			got:  Internal(InternalConfig{ErrorHandler: errorHandler, Authenticator: testInternalAuthenticator{}}),
			want: 2,
		},
		{
			name: "mutating",
			got:  Mutating(MutatingConfig{ErrorHandler: errorHandler, Authorizer: testAuthorizer{}}),
			want: 1,
		},
		{
			name: "protected",
			got:  Protected(ProtectedConfig{ErrorHandler: errorHandler, Authenticator: testAuthenticator{}}),
			want: 2,
		},
		{
			name: "public",
			got:  Public(PublicConfig{ErrorHandler: errorHandler}),
			want: 3,
		},
		{
			name: "webhook",
			got: Webhook(WebhookConfig{
				ErrorHandler: errorHandler,
				Signature: middleware.WebhookSignatureConfig{
					Verifier: testVerifier{},
				},
			}),
			want: 2,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Len(t, tt.got, tt.want)
		})
	}
}

func TestGatewayAddsRateLimitMiddlewareWhenConfigured(t *testing.T) {
	t.Parallel()

	chain := Gateway(GatewayConfig{
		Base:              BaseConfig{},
		RateLimitRequests: 10,
		RateLimitWindow:   time.Second,
	}, GatewayDeps{ErrorHandler: httphandler.NewErrorHandler(nil)})

	require.Len(t, chain, 12)
}

type testIdempotencyStore struct{}

func (testIdempotencyStore) Start(rctx context.Context, key string, fingerprint string, ttl time.Duration) (middleware.IdempotencyStartResult, error) {
	return middleware.IdempotencyStartResult{State: middleware.IdempotencyStarted}, nil
}

func (testIdempotencyStore) Finish(rctx context.Context, key string, fingerprint string, record middleware.IdempotencyRecord, ttl time.Duration) error {
	return nil
}

func (testIdempotencyStore) Fail(rctx context.Context, key string, fingerprint string) error {
	return nil
}
