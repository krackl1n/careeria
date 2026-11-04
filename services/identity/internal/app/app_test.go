package app

import (
	"bytes"
	"testing"
	"time"

	"github.com/krackl1n/careeria/closer"
	grpcserver "github.com/krackl1n/careeria/grpc/server"
	sharedlogger "github.com/krackl1n/careeria/logger"
	sharedmetrics "github.com/krackl1n/careeria/metrics"
	"github.com/krackl1n/careeria/postgres"
	internalconfig "github.com/krackl1n/careeria/services/identity/internal/config"
	accesstokenservice "github.com/krackl1n/careeria/services/identity/internal/service/accesstoken"
	contactverificationservice "github.com/krackl1n/careeria/services/identity/internal/service/contactverification"
	idtokenservice "github.com/krackl1n/careeria/services/identity/internal/service/idtoken"
	jwtsigner "github.com/krackl1n/careeria/services/identity/internal/signer/jwt"
	oauthusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/oauth"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestNewValidatesDependencies(t *testing.T) {
	_, err := New(nil)
	require.EqualError(t, err, "application dependencies are required")
	_, err = New(&Dependencies{})
	require.EqualError(t, err, "application logger is required")
	deps := testDependencies()
	deps.RedisClient = nil
	_, err = New(deps)
	require.EqualError(t, err, "application Redis client is required")
}

func TestLazyGraphRegistersEveryIdentityService(t *testing.T) {
	application, err := New(testDependencies())
	require.NoError(t, err)
	first, err := application.OAuth()
	require.NoError(t, err)
	second, err := application.OAuth()
	require.NoError(t, err)
	require.Same(t, first, second)
	server, err := application.GRPCServer()
	require.NoError(t, err)
	services := server.GRPC().GetServiceInfo()
	require.Contains(t, services, "careeria.identity.v1.AuthenticationService")
	require.Contains(t, services, "careeria.identity.v1.RegistrationService")
	require.Contains(t, services, "careeria.identity.v1.OAuthService")
	require.Contains(t, services, "careeria.identity.v1.IdentityMetadataService")
	require.Contains(t, services, "careeria.identity.admin.v1.OAuthClientAdminService")
	require.Contains(t, services, "careeria.identity.admin.v1.SigningKeyAdminService")
}

func testDependencies() *Dependencies {
	return &Dependencies{Config: internalconfig.Config{
		GRPCServer:                grpcserver.Config{Addr: "127.0.0.1:0"},
		Metrics:                   sharedmetrics.Config{Enabled: false},
		AccessToken:               accesstokenservice.Config{Issuer: "https://identity.example", Audience: []string{"careeria-api"}, Algorithm: "EdDSA", Lifetime: time.Minute},
		IDToken:                   idtokenservice.Config{Issuer: "https://identity.example", Algorithm: "EdDSA", Lifetime: time.Minute},
		ContactVerification:       contactverificationservice.Config{Lifetime: time.Minute, MaxAttempts: 5},
		JWTSigner:                 jwtsigner.Config{KeyID: "identity-local-1", PrivateKeyRef: "local://access-token-signing-key", KeyFile: "../../configs/access-token-signing-key.pem"},
		OAuth:                     oauthusecase.Config{AuthorizationLifetime: time.Minute, CodeLifetime: time.Minute, SessionIdleLifetime: time.Minute, SessionLifetime: time.Hour, RefreshTokenLifetime: time.Minute, RefreshFamilyLifetime: time.Hour},
		AuthenticationRedirectURI: "https://identity.example/login",
	}, Logger: sharedlogger.New(sharedlogger.Config{}, &bytes.Buffer{}), Closer: closer.New(), PostgresClient: &postgres.Client{}, RedisClient: goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:0"})}
}
