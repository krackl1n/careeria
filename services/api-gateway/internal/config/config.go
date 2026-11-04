package config

import (
	"fmt"

	grpcclient "github.com/krackl1n/careeria/grpc/client"
	grpcclientinterceptor "github.com/krackl1n/careeria/grpc/client/interceptor"
	httppreset "github.com/krackl1n/careeria/http/middleware/preset"
	httpserver "github.com/krackl1n/careeria/http/server"
	sharedlogger "github.com/krackl1n/careeria/logger"
	sharedmetrics "github.com/krackl1n/careeria/metrics"
	"github.com/krackl1n/careeria/services/api-gateway/internal/authn"
	swaggertransport "github.com/krackl1n/careeria/services/api-gateway/internal/transport/http/swagger"
	"github.com/krackl1n/careeria/shutdown"
	sharedtracing "github.com/krackl1n/careeria/tracing"
)

type Config struct {
	HTTP httpserver.Config

	Logger  sharedlogger.Config
	Metrics sharedmetrics.Config
	Tracing sharedtracing.Config

	Gateway        httppreset.GatewayConfig
	Swagger        swaggertransport.Config
	IdentityGRPC   grpcclient.Config
	Authentication authn.AuthConfig

	GRPCClientMetrics grpcclientinterceptor.MetricsConfig

	Shutdown shutdown.Config
}

func (c Config) Validate() error {
	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("validate HTTP server config: %w", err)
	}

	if err := c.Logger.Validate(); err != nil {
		return fmt.Errorf("validate logger config: %w", err)
	}

	if err := c.Shutdown.Validate(); err != nil {
		return fmt.Errorf("validate shutdown config: %w", err)
	}
	if err := c.Metrics.Validate(); err != nil {
		return fmt.Errorf("validate metrics config: %w", err)
	}
	if err := c.Tracing.Validate(); err != nil {
		return fmt.Errorf("validate tracing config: %w", err)
	}

	if err := c.Gateway.Validate(); err != nil {
		return fmt.Errorf("validate gateway config: %w", err)
	}
	if err := c.Swagger.Validate(); err != nil {
		return fmt.Errorf("validate swagger config: %w", err)
	}

	if err := c.IdentityGRPC.Validate(); err != nil {
		return fmt.Errorf("validate identity gRPC client config: %w", err)
	}
	if err := c.Authentication.Validate(); err != nil {
		return fmt.Errorf("validate authentication config: %w", err)
	}

	return nil
}
