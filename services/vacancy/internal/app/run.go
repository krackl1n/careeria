package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	openfga "github.com/krackl1n/careeria/authz/openfga/grpc"
	grpcclient "github.com/krackl1n/careeria/grpc/client"
	"github.com/krackl1n/careeria/logger"
	"github.com/krackl1n/careeria/postgres"
	"github.com/krackl1n/careeria/runtime"
	"github.com/krackl1n/careeria/services/vacancy/internal/config"
	vacancyopenfga "github.com/krackl1n/careeria/services/vacancy/internal/integrations/openfga"
	"github.com/krackl1n/careeria/shutdown"
)

func Run(parent context.Context) error {
	ctx, cancel := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(cfg.Logger, nil)
	client, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("initialize postgres client: %w", err)
	}
	defer client.Close()

	connection, err := grpcclient.New(cfg.AuthorizationClient)
	if err != nil {
		return fmt.Errorf("initialize Authorization connection: %w", err)
	}
	defer connection.Close()
	fgaClient, err := openfga.New(cfg.Authorization, connection)
	if err != nil {
		return fmt.Errorf("initialize Authorization client: %w", err)
	}
	authorizer, err := vacancyopenfga.NewAuthorizer(fgaClient)
	if err != nil {
		return fmt.Errorf("initialize authorization adapter: %w", err)
	}
	server, err := newGRPCServer(cfg, log, client, authorizer)
	if err != nil {
		return fmt.Errorf("initialize gRPC server: %w", err)
	}

	group, err := runtime.Start(server)
	if err != nil {
		return fmt.Errorf("start vacancy service: %w", err)
	}
	log.Info("starting vacancy service", "addr", cfg.GRPCServer.Addr)

	var serveErr error
	select {
	case result := <-group.Results():
		serveErr = result.Err
	case <-ctx.Done():
	}

	return errors.Join(serveErr, shutdown.Run(context.Background(), cfg.Shutdown, server))
}
