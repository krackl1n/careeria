package app

import (
	"fmt"

	identityadminv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/admin/v1"
	identityv1 "github.com/krackl1n/careeria/contracts/gen/careeria/identity/v1"
	grpcpreset "github.com/krackl1n/careeria/grpc/preset"
	grpcserver "github.com/krackl1n/careeria/grpc/server"
	"google.golang.org/grpc"
)

func (a *App) GRPCServer() (*grpcserver.Server, error) {
	if a.grpcServer != nil {
		return a.grpcServer, nil
	}
	metrics, err := a.GRPCMetrics()
	if err != nil {
		return nil, err
	}
	preset := grpcpreset.Server(grpcpreset.ServerConfig{Logger: a.dependencies.Logger, Metrics: metrics, ErrorResolver: a.ErrorResolver()})
	server, err := grpcserver.New(a.dependencies.Config.GRPCServer, grpcserver.WithUnaryInterceptors(preset.Unary...), grpcserver.WithStreamInterceptors(preset.Stream...), grpcserver.WithServerOptions(grpc.StatsHandler(preset.StatsHandler)))
	if err != nil {
		return nil, err
	}
	authentication, err := a.AuthenticationHandler()
	if err != nil {
		return nil, fmt.Errorf("initialize authentication handler: %w", err)
	}
	registration, err := a.RegistrationHandler()
	if err != nil {
		return nil, fmt.Errorf("initialize registration handler: %w", err)
	}
	oauth, err := a.OAuthHandler()
	if err != nil {
		return nil, fmt.Errorf("initialize OAuth handler: %w", err)
	}
	metadata, err := a.MetadataHandler()
	if err != nil {
		return nil, fmt.Errorf("initialize metadata handler: %w", err)
	}
	signingKeyAdmin, err := a.SigningKeyAdminHandler()
	if err != nil {
		return nil, fmt.Errorf("initialize signing key admin handler: %w", err)
	}
	identityv1.RegisterAuthenticationServiceServer(server.GRPC(), authentication)
	identityv1.RegisterRegistrationServiceServer(server.GRPC(), registration)
	identityv1.RegisterOAuthServiceServer(server.GRPC(), oauth)
	identityv1.RegisterIdentityMetadataServiceServer(server.GRPC(), metadata)
	identityadminv1.RegisterOAuthClientAdminServiceServer(server.GRPC(), a.OAuthClientAdminHandler())
	identityadminv1.RegisterSigningKeyAdminServiceServer(server.GRPC(), signingKeyAdmin)
	a.grpcServer = server
	return server, nil
}
