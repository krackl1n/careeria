package app

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/krackl1n/careeria/closer"
	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	contractsopenapi "github.com/krackl1n/careeria/contracts/openapi"
	"github.com/krackl1n/careeria/ctxmeta"
	grpcclient "github.com/krackl1n/careeria/grpc/client"
	grpchttpresponse "github.com/krackl1n/careeria/grpc/httpresponse"
	httpmiddleware "github.com/krackl1n/careeria/http/middleware"
	httpserver "github.com/krackl1n/careeria/http/server"
	sharedlogger "github.com/krackl1n/careeria/logger"
	sharedmetrics "github.com/krackl1n/careeria/metrics"
	"github.com/krackl1n/careeria/services/api-gateway/internal/authn"
	internalconfig "github.com/krackl1n/careeria/services/api-gateway/internal/config"
	swaggertransport "github.com/krackl1n/careeria/services/api-gateway/internal/transport/http/swagger"
)

var swaggerConfig = swaggertransport.Config{
	Enabled:           true,
	Title:             "Careeria API",
	Path:              "/swagger/",
	SpecificationPath: "/openapi/careeria.swagger.json",
}

var identityGRPCConfig = grpcclient.Config{
	Target:   "127.0.0.1:9095",
	Insecure: true,
}

var authenticationConfig = authn.AuthConfig{
	Issuer:   "https://identity.example",
	Audience: []string{"careeria-api"},
	JWKSCache: authn.JWKSCacheConfig{
		TTL:             time.Minute,
		RefreshTimeout:  5 * time.Second,
		RefreshCooldown: 10 * time.Second,
	},
}

func TestNew(t *testing.T) {
	_, err := New(nil)
	require.Error(t, err)
}

func TestRuntimesInitializesHTTPServer(t *testing.T) {
	dependencies := &Dependencies{
		Config: internalconfig.Config{
			HTTP: httpserver.Config{
				Addr: "127.0.0.1:0",
			},
			IdentityGRPC:   identityGRPCConfig,
			Authentication: authenticationConfig,
		},
		Logger: sharedlogger.New(sharedlogger.Config{}, &bytes.Buffer{}),
		Closer: closer.New(),
	}

	application, err := New(dependencies)
	require.NoError(t, err)

	runtimes, err := application.Runtimes(t.Context())
	require.NoError(t, err)
	require.Len(t, runtimes, 2)
	require.NotNil(t, application.httpServer)

	require.NoError(t, dependencies.Closer.Close())
}

func TestRuntimesInitializesMetricsServer(t *testing.T) {
	dependencies := &Dependencies{
		Config: internalconfig.Config{
			HTTP: httpserver.Config{
				Addr: "127.0.0.1:0",
			},
			Metrics: sharedmetrics.Config{
				Enabled:           true,
				Addr:              "127.0.0.1:0",
				Namespace:         "careeria",
				ReadHeaderTimeout: 5 * time.Second,
			},
			IdentityGRPC:   identityGRPCConfig,
			Authentication: authenticationConfig,
		},
		Logger: sharedlogger.New(sharedlogger.Config{}, &bytes.Buffer{}),
		Closer: closer.New(),
	}

	application, err := New(dependencies)
	require.NoError(t, err)

	runtimes, err := application.Runtimes(t.Context())
	require.NoError(t, err)
	require.Len(t, runtimes, 2)
	require.NotNil(t, application.metrics)

	require.NoError(t, dependencies.Closer.Close())
}

func TestRuntimesReturnsHTTPInitializationError(t *testing.T) {
	dependencies := &Dependencies{
		Config: internalconfig.Config{
			HTTP: httpserver.Config{
				Addr: "127.0.0.1:0",
			},
			IdentityGRPC: grpcclient.Config{
				Target: "",
			},
			Authentication: authenticationConfig,
		},
		Logger: sharedlogger.New(sharedlogger.Config{}, &bytes.Buffer{}),
		Closer: closer.New(),
	}

	application, err := New(dependencies)
	require.NoError(t, err)

	_, err = application.Runtimes(t.Context())
	require.Error(t, err)
}

func TestHTTPServerAppliesGatewayPreset(t *testing.T) {
	dependencies := &Dependencies{
		Config: internalconfig.Config{
			HTTP: httpserver.Config{
				Addr: "127.0.0.1:0",
			},
			IdentityGRPC:   identityGRPCConfig,
			Authentication: authenticationConfig,
		},
		Logger: sharedlogger.New(sharedlogger.Config{}, &bytes.Buffer{}),
		Closer: closer.New(),
	}

	application, err := New(dependencies)
	require.NoError(t, err)

	server, err := application.HTTPServer(t.Context())
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	server.HTTP().Handler.ServeHTTP(recorder, request)

	require.NotEmpty(t, recorder.Header().Get("X-Request-Id"))
	require.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
}

func TestHTTPServerServesSwagger(t *testing.T) {
	dependencies := &Dependencies{
		Config: internalconfig.Config{
			HTTP: httpserver.Config{
				Addr: "127.0.0.1:0",
			},
			Swagger:        swaggerConfig,
			IdentityGRPC:   identityGRPCConfig,
			Authentication: authenticationConfig,
		},
		Logger: sharedlogger.New(sharedlogger.Config{}, &bytes.Buffer{}),
		Closer: closer.New(),
	}

	application, err := New(dependencies)
	require.NoError(t, err)

	server, err := application.HTTPServer(t.Context())
	require.NoError(t, err)

	swaggerRecorder := httptest.NewRecorder()
	swaggerRequest := httptest.NewRequest(http.MethodGet, "/swagger/", nil)
	server.HTTP().Handler.ServeHTTP(swaggerRecorder, swaggerRequest)
	require.Equal(t, http.StatusOK, swaggerRecorder.Code)

	specRecorder := httptest.NewRecorder()
	specRequest := httptest.NewRequest(http.MethodGet, "/openapi/careeria.swagger.json", nil)
	server.HTTP().Handler.ServeHTTP(specRecorder, specRequest)
	require.Equal(t, http.StatusOK, specRecorder.Code)
	require.Equal(t, "application/json; charset=utf-8", specRecorder.Header().Get("Content-Type"))
	require.Equal(t, contractsopenapi.Specification(), specRecorder.Body.Bytes())
}

func TestGatewayMuxInitializesOnce(t *testing.T) {
	dependencies := &Dependencies{
		Config: internalconfig.Config{
			IdentityGRPC:   identityGRPCConfig,
			Authentication: authenticationConfig,
		},
		Logger: sharedlogger.New(sharedlogger.Config{}, &bytes.Buffer{}),
		Closer: closer.New(),
	}

	application, err := New(dependencies)
	require.NoError(t, err)

	first, err := application.GatewayMux(t.Context())
	require.NoError(t, err)

	second, err := application.GatewayMux(t.Context())
	require.NoError(t, err)
	require.Same(t, first, second)
}

func TestGatewayMuxRegistersEveryIdentityService(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	identityv1.RegisterOAuthServiceServer(server, &unimplementedOAuthServer{})
	identityv1.RegisterAuthenticationServiceServer(server, &unimplementedAuthenticationServer{})
	identityv1.RegisterIdentitySessionServiceServer(server, &unimplementedIdentitySessionServer{})
	identityv1.RegisterIdentityMetadataServiceServer(server, &unimplementedIdentityMetadataServer{})
	identityv1.RegisterRegistrationServiceServer(server, &unimplementedRegistrationServer{})
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)

	dependencies := &Dependencies{
		Config: internalconfig.Config{
			Authentication: authenticationConfig,
			IdentityGRPC: grpcclient.Config{
				Target:   listener.Addr().String(),
				Insecure: true,
			},
		},
		Logger: sharedlogger.New(sharedlogger.Config{}, &bytes.Buffer{}),
		Closer: closer.New(),
	}
	t.Cleanup(func() {
		require.NoError(t, dependencies.Closer.Close())
	})

	application, err := New(dependencies)
	require.NoError(t, err)
	application.authenticator = authenticatedUser()
	mux, err := application.GatewayMux(t.Context())
	require.NoError(t, err)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{
			name:   "OAuth authorization",
			method: http.MethodGet,
			path:   "/oauth2/authorize?response_type=code&client_id=web&redirect_uri=https://example.com/callback&scope=openid",
		},
		{
			name:   "Authentication methods",
			method: http.MethodGet,
			path:   "/api/v1/authentication/methods",
		},
		{
			name:   "Identity session",
			method: http.MethodGet,
			path:   "/api/v1/session",
		},
		{
			name:   "Identity metadata configuration",
			method: http.MethodGet,
			path:   "/.well-known/openid-configuration",
		},
		{
			name:   "Registration",
			method: http.MethodPost,
			path:   "/api/v1/registration",
			body:   `{"email":"test@example.com","password":"password123","authorization_transaction_id":"018f68a5-0ef3-7b8b-9cf8-3f31da58de09","profile":{"family_name":"Doe","given_name":"John"}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			var bodyReader *bytes.Reader
			if test.body != "" {
				bodyReader = bytes.NewReader([]byte(test.body))
			}
			var request *http.Request
			if bodyReader != nil {
				request = httptest.NewRequest(test.method, test.path, bodyReader)
				request.Header.Set("Content-Type", "application/json")
			} else {
				request = httptest.NewRequest(test.method, test.path, nil)
			}
			request.Header.Set("Authorization", "Bearer test-token")
			mux.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusNotImplemented, recorder.Code)
			require.JSONEq(t, `{
				"error": {
					"code": "not_implemented",
					"message": "Not Implemented"
				}
			}`, recorder.Body.String())
		})
	}
}

func TestGatewayMuxBuildsRedirectFromGRPCMetadata(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	identityv1.RegisterOAuthServiceServer(server, &redirectOAuthServer{})
	identityv1.RegisterAuthenticationServiceServer(server, &unimplementedAuthenticationServer{})
	identityv1.RegisterIdentitySessionServiceServer(server, &unimplementedIdentitySessionServer{})
	identityv1.RegisterIdentityMetadataServiceServer(server, &unimplementedIdentityMetadataServer{})
	identityv1.RegisterRegistrationServiceServer(server, &unimplementedRegistrationServer{})
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)

	dependencies := &Dependencies{
		Config: internalconfig.Config{
			Authentication: authenticationConfig,
			IdentityGRPC: grpcclient.Config{
				Target:   listener.Addr().String(),
				Insecure: true,
			},
		},
		Logger: sharedlogger.New(sharedlogger.Config{}, &bytes.Buffer{}),
		Closer: closer.New(),
	}
	t.Cleanup(func() {
		require.NoError(t, dependencies.Closer.Close())
	})

	application, err := New(dependencies)
	require.NoError(t, err)
	application.authenticator = authenticatedUser()
	mux, err := application.GatewayMux(t.Context())
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/oauth2/authorize?response_type=code&client_id=web&redirect_uri=https://example.com/callback&scope=openid", nil)
	mux.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusFound, recorder.Code)
	require.Equal(t, "https://identity.example/login", recorder.Header().Get("Location"))
	require.Empty(t, recorder.Header().Get("Grpc-Metadata-Location"))
	require.Empty(t, recorder.Header().Get("Grpc-Metadata-Status-Code"))
}

func TestIdentityIntegrationInitializesClientsOnce(t *testing.T) {
	dependencies := &Dependencies{
		Config: internalconfig.Config{IdentityGRPC: identityGRPCConfig},
		Logger: sharedlogger.New(sharedlogger.Config{}, &bytes.Buffer{}),
		Closer: closer.New(),
	}

	application, err := New(dependencies)
	require.NoError(t, err)

	first, err := application.IdentityIntegration()
	require.NoError(t, err)
	require.NotNil(t, first.OAuth)
	require.NotNil(t, first.Authentication)
	require.NotNil(t, first.Session)
	require.NotNil(t, first.Metadata)
	require.NotNil(t, first.Registration)

	second, err := application.IdentityIntegration()
	require.NoError(t, err)
	require.Same(t, first, second)

	require.NoError(t, dependencies.Closer.Close())
}

func authenticatedUser() httpmiddleware.Authenticator {
	return httpmiddleware.AuthenticatorFunc(func(*http.Request) (ctxmeta.Principal, error) {
		return ctxmeta.Principal{SubjectID: "018f68a5-0ef3-7b8b-9cf8-3f31da58de09"}, nil
	})
}

type unimplementedOAuthServer struct {
	identityv1.UnimplementedOAuthServiceServer
}

type redirectOAuthServer struct {
	identityv1.UnimplementedOAuthServiceServer
}

func (*redirectOAuthServer) Authorize(ctx context.Context, _ *identityv1.AuthorizeRequest) (*identityv1.AuthorizeResponse, error) {
	const location = "https://identity.example/login"
	if err := grpchttpresponse.SetRedirect(ctx, http.StatusFound, location); err != nil {
		return nil, err
	}
	return &identityv1.AuthorizeResponse{RedirectUri: location}, nil
}

type unimplementedAuthenticationServer struct {
	identityv1.UnimplementedAuthenticationServiceServer
}

type unimplementedIdentitySessionServer struct {
	identityv1.UnimplementedIdentitySessionServiceServer
}

type unimplementedIdentityMetadataServer struct {
	identityv1.UnimplementedIdentityMetadataServiceServer
}

type unimplementedRegistrationServer struct {
	identityv1.UnimplementedRegistrationServiceServer
}
