package config

import (
	"fmt"

	openfgagrpc "github.com/krackl1n/careeria/authz/openfga/grpc"
	grpcclient "github.com/krackl1n/careeria/grpc/client"
	grpcserver "github.com/krackl1n/careeria/grpc/server"
	"github.com/krackl1n/careeria/logger"
	"github.com/krackl1n/careeria/postgres"
	"github.com/krackl1n/careeria/shutdown"
)

type Config struct {
	Logger              logger.Config
	GRPCServer          grpcserver.Config
	Shutdown            shutdown.Config
	Postgres            postgres.Config
	Authorization       openfgagrpc.Config
	AuthorizationClient grpcclient.Config
}

func (c Config) Validate() error {
	if err := c.Logger.Validate(); err != nil {
		return fmt.Errorf("validate logger config: %w", err)
	}
	if err := c.GRPCServer.Validate(); err != nil {
		return fmt.Errorf("validate gRPC server config: %w", err)
	}
	if err := c.Shutdown.Validate(); err != nil {
		return fmt.Errorf("validate shutdown config: %w", err)
	}
	if err := c.Postgres.Validate(); err != nil {
		return fmt.Errorf("validate postgres config: %w", err)
	}
	if c.Postgres.Connection.Password == "" {
		return fmt.Errorf("VACANCY_POSTGRES_CONNECTION_PASSWORD is required")
	}
	return nil
}
