package app

import (
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/krackl1n/careeria/clock"
	pb "github.com/krackl1n/careeria/contracts/gen/careeria/vacancy/v1"
	"github.com/krackl1n/careeria/grpc/interceptor"
	grpcserver "github.com/krackl1n/careeria/grpc/server"
	"github.com/krackl1n/careeria/idgen"
	"github.com/krackl1n/careeria/postgres"
	"github.com/krackl1n/careeria/services/vacancy/internal/config"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
	outboxrepo "github.com/krackl1n/careeria/services/vacancy/internal/repository/outbox"
	vacancyrepo "github.com/krackl1n/careeria/services/vacancy/internal/repository/vacancy"
	transport "github.com/krackl1n/careeria/services/vacancy/internal/transport/grpc"
	vacancyhandler "github.com/krackl1n/careeria/services/vacancy/internal/transport/grpc/vacancy"
	commands "github.com/krackl1n/careeria/services/vacancy/internal/usecase/vacancy"
	queries "github.com/krackl1n/careeria/services/vacancy/internal/usecase/vacancyquery"
	"github.com/krackl1n/careeria/txmanager"
)

func newGRPCServer(cfg config.Config, logger *slog.Logger, client *postgres.Client, authorizer domain.Authorizer) (*grpcserver.Server, error) {
	if authorizer == nil {
		return nil, fmt.Errorf("authorizer is required")
	}
	repo := vacancyrepo.NewPostgresRepository(client)
	commandService, err := commands.NewService(repo, outboxrepo.NewPostgresRepository(client), txmanager.New[pgx.Tx](client), clock.New(), idgen.NewUUID())
	if err != nil {
		return nil, fmt.Errorf("create vacancy command service: %w", err)
	}
	queryService, err := queries.NewService(repo)
	if err != nil {
		return nil, fmt.Errorf("create vacancy query service: %w", err)
	}
	server, err := grpcserver.New(cfg.GRPCServer, grpcserver.WithUnaryInterceptors(
		interceptor.UnaryRecovery(logger, nil),
		interceptor.UnaryRequestID(interceptor.RequestIDConfig{}),
		interceptor.UnaryLogging(logger),
		interceptor.UnaryErrorMapping(logger, transport.ErrorResolver{}),
		interceptor.UnaryValidation(),
		transport.UnaryPrincipal,
	), grpcserver.WithStreamInterceptors(
		interceptor.StreamRecovery(logger, nil),
		interceptor.StreamRequestID(interceptor.RequestIDConfig{}),
		interceptor.StreamLogging(logger),
		interceptor.StreamErrorMapping(logger, transport.ErrorResolver{}),
		interceptor.StreamValidation(),
	))
	if err != nil {
		return nil, err
	}
	handler := vacancyhandler.NewHandler(&authorizedVacancyService{commands: commandService, queries: queryService, authorizer: authorizer})
	pb.RegisterVacancyServiceServer(server.GRPC(), handler)
	pb.RegisterVacancyQueryServiceServer(server.GRPC(), handler)
	return server, nil
}
