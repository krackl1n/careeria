package app

import (
	"context"
	"errors"
	"fmt"

	gatewayruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	contractsopenapi "github.com/krackl1n/careeria/contracts/openapi"
	grpcclientinterceptor "github.com/krackl1n/careeria/grpc/client/interceptor"
	httpmiddleware "github.com/krackl1n/careeria/http/middleware"
	httppreset "github.com/krackl1n/careeria/http/middleware/preset"
	httpserver "github.com/krackl1n/careeria/http/server"
	sharedmetrics "github.com/krackl1n/careeria/metrics"
	sharedruntime "github.com/krackl1n/careeria/runtime"
	"github.com/krackl1n/careeria/services/api-gateway/internal/authn"
	grpcintegration "github.com/krackl1n/careeria/services/api-gateway/internal/integrations/grpc"
	identityintegration "github.com/krackl1n/careeria/services/api-gateway/internal/integrations/identity"
	httptransport "github.com/krackl1n/careeria/services/api-gateway/internal/transport/http"
	gatewaytransport "github.com/krackl1n/careeria/services/api-gateway/internal/transport/http/grpcgateway"
)

type App struct {
	dependencies *Dependencies

	gatewayMux *gatewayruntime.ServeMux

	httpServer        *httpserver.Server
	metrics           *sharedmetrics.Component
	grpcClientMetrics *grpcclientinterceptor.Metrics
	grpcDialer        *grpcintegration.Dialer

	authenticator httpmiddleware.Authenticator

	identityIntegration *identityintegration.Clients
}

func New(dependencies *Dependencies) (*App, error) {
	if dependencies == nil {
		return nil, errors.New("application dependencies are required")
	}
	if dependencies.Logger == nil {
		return nil, errors.New("application logger is required")
	}
	if dependencies.Closer == nil {
		return nil, errors.New("application closer is required")
	}

	return &App{
		dependencies: dependencies,
	}, nil
}

func (a *App) Runtimes(ctx context.Context) ([]sharedruntime.Component, error) {
	metrics, err := a.Metrics()
	if err != nil {
		return nil, fmt.Errorf("initialize metrics: %w", err)
	}

	server, err := a.HTTPServer(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize HTTP server: %w", err)
	}

	return []sharedruntime.Component{metrics, server}, nil
}

func (a *App) HTTPServer(ctx context.Context) (*httpserver.Server, error) {
	if a.httpServer != nil {
		return a.httpServer, nil
	}

	mux, err := a.GatewayMux(ctx)
	if err != nil {
		return nil, err
	}

	metrics, err := a.Metrics()
	if err != nil {
		return nil, fmt.Errorf("initialize metrics: %w", err)
	}

	router, err := httptransport.NewRouter(httptransport.RouterDeps{
		SwaggerConfig:        a.dependencies.Config.Swagger,
		GatewayConfig:        a.dependencies.Config.Gateway,
		GRPCGateway:          mux,
		SwaggerSpecification: contractsopenapi.Specification(),
		MiddlewarePresetDeps: httppreset.GatewayDeps{
			Logger:     a.dependencies.Logger,
			Registerer: metrics.Registry(),
		},
	})
	if err != nil {
		return nil, err
	}

	server, err := httpserver.New(
		a.dependencies.Config.HTTP,
		router,
	)
	if err != nil {
		return nil, err
	}

	a.httpServer = server
	return a.httpServer, nil
}

func (a *App) Authenticator() (httpmiddleware.Authenticator, error) {
	if a.authenticator != nil {
		return a.authenticator, nil
	}

	identityClients, err := a.IdentityIntegration()
	if err != nil {
		return nil, err
	}
	jwksCache, err := authn.NewJWKSCache(
		identityClients,
		a.dependencies.Config.Authentication.JWKSCache,
	)
	if err != nil {
		return nil, fmt.Errorf("initialize JWKS cache: %w", err)
	}

	authenticator, err := authn.NewJWT(
		a.dependencies.Config.Authentication,
		jwksCache.ResolvePublicKey,
	)
	if err != nil {
		return nil, err
	}

	a.authenticator = authenticator
	return authenticator, nil
}

func (a *App) GatewayMux(ctx context.Context) (*gatewayruntime.ServeMux, error) {
	if a.gatewayMux != nil {
		return a.gatewayMux, nil
	}
	if ctx == nil {
		return nil, errors.New("gateway context is required")
	}

	identityClients, err := a.IdentityIntegration()
	if err != nil {
		return nil, fmt.Errorf("initialize identity integration: %w", err)
	}
	authenticator, err := a.Authenticator()
	if err != nil {
		return nil, fmt.Errorf("initialize gateway authenticator: %w", err)
	}

	mux, err := gatewaytransport.NewMux(ctx, gatewaytransport.MuxDeps{
		Logger:        a.dependencies.Logger,
		Authenticator: authenticator,
		Identity:      identityClients,
	})
	if err != nil {
		return nil, err
	}

	a.gatewayMux = mux
	return a.gatewayMux, nil
}

func (a *App) GRPCClientMetrics() (*grpcclientinterceptor.Metrics, error) {
	if a.grpcClientMetrics != nil {
		return a.grpcClientMetrics, nil
	}

	metricsComponent, err := a.Metrics()
	if err != nil {
		return nil, fmt.Errorf("initialize metrics: %w", err)
	}
	metrics, err := grpcclientinterceptor.NewMetrics(
		a.dependencies.Config.GRPCClientMetrics,
		metricsComponent,
	)
	if err != nil {
		return nil, err
	}
	a.grpcClientMetrics = metrics
	return a.grpcClientMetrics, nil
}

func (a *App) IntegrationGRPCDialer() (*grpcintegration.Dialer, error) {
	if a.grpcDialer != nil {
		return a.grpcDialer, nil
	}

	metrics, err := a.GRPCClientMetrics()
	if err != nil {
		return nil, fmt.Errorf("initialize gRPC client metrics: %w", err)
	}

	dialer, err := grpcintegration.NewDialer(metrics)
	if err != nil {
		return nil, err
	}

	a.grpcDialer = dialer
	return a.grpcDialer, nil
}

func (a *App) Metrics() (*sharedmetrics.Component, error) {
	if a.metrics != nil {
		return a.metrics, nil
	}

	metrics, err := sharedmetrics.New(a.dependencies.Config.Metrics)
	if err != nil {
		return nil, err
	}

	a.metrics = metrics
	return a.metrics, nil
}

func (a *App) IdentityIntegration() (*identityintegration.Clients, error) {
	if a.identityIntegration != nil {
		return a.identityIntegration, nil
	}

	dialer, err := a.IntegrationGRPCDialer()
	if err != nil {
		return nil, err
	}

	connection, err := dialer.Dial(a.dependencies.Config.IdentityGRPC, "identity")
	if err != nil {
		return nil, err
	}
	a.dependencies.Closer.Add(connection)

	a.identityIntegration = identityintegration.New(connection)
	return a.identityIntegration, nil
}
