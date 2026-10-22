package config

import (
	"fmt"
	"strings"

	grpcserver "github.com/krackl1n/careeria/grpc/server"
	sharedlogger "github.com/krackl1n/careeria/logger"
	sharedmetrics "github.com/krackl1n/careeria/metrics"
	sharedoutbox "github.com/krackl1n/careeria/outbox"
	"github.com/krackl1n/careeria/postgres"
	sharedshutdown "github.com/krackl1n/careeria/shutdown"
	sharedtracing "github.com/krackl1n/careeria/tracing"
)

type Config struct {
	Logger      sharedlogger.Config
	Metrics     sharedmetrics.Config
	Tracing     sharedtracing.Config
	Postgres    postgres.Config
	GRPCServer  grpcserver.Config
	Shutdown    sharedshutdown.Config
	HiringTopic string `mapstructure:"hiring_topic"`
	Outbox      sharedoutbox.Config
}

func (c Config) Validate() error {
	if err := c.Logger.Validate(); err != nil {
		return fmt.Errorf("validate logger config: %w", err)
	}
	if err := c.Metrics.Validate(); err != nil {
		return fmt.Errorf("validate metrics config: %w", err)
	}
	if err := c.Tracing.Validate(); err != nil {
		return fmt.Errorf("validate tracing config: %w", err)
	}
	if err := c.Postgres.Validate(); err != nil {
		return fmt.Errorf("validate postgres config: %w", err)
	}
	if err := c.GRPCServer.Validate(); err != nil {
		return fmt.Errorf("validate gRPC server config: %w", err)
	}
	if err := c.Shutdown.Validate(); err != nil {
		return fmt.Errorf("validate shutdown config: %w", err)
	}
	if strings.TrimSpace(c.HiringTopic) == "" {
		return fmt.Errorf("hiring topic is required")
	}
	if err := c.Outbox.Validate(); err != nil {
		return fmt.Errorf("validate outbox config: %w", err)
	}

	return nil
}
