package config

import (
	"fmt"
	"strings"

	grpcserver "github.com/krackl1n/careeria/grpc/server"
	sharedkafka "github.com/krackl1n/careeria/kafka"
	sharedlogger "github.com/krackl1n/careeria/logger"
	sharedmetrics "github.com/krackl1n/careeria/metrics"
	sharedoutbox "github.com/krackl1n/careeria/outbox"
	"github.com/krackl1n/careeria/postgres"
	accesstokenservice "github.com/krackl1n/careeria/services/identity/internal/service/accesstoken"
	contactverificationservice "github.com/krackl1n/careeria/services/identity/internal/service/contactverification"
	idtokenservice "github.com/krackl1n/careeria/services/identity/internal/service/idtoken"
	jwtsigner "github.com/krackl1n/careeria/services/identity/internal/signer/jwt"
	oauthusecase "github.com/krackl1n/careeria/services/identity/internal/usecase/oauth"
	sharedshutdown "github.com/krackl1n/careeria/shutdown"
	sharedtracing "github.com/krackl1n/careeria/tracing"
)

type Config struct {
	Logger                    sharedlogger.Config
	Metrics                   sharedmetrics.Config
	Tracing                   sharedtracing.Config
	Postgres                  postgres.Config
	GRPCServer                grpcserver.Config
	Shutdown                  sharedshutdown.Config
	Redis                     RedisConfig
	AuthenticationRedirectURI string `mapstructure:"authentication_redirect_uri"`
	KafkaProducer             sharedkafka.ProducerConfig
	IdentityTopic             string `mapstructure:"identity_topic"`
	Outbox                    sharedoutbox.Config

	AccessToken         accesstokenservice.Config
	IDToken             idtokenservice.Config
	ContactVerification contactverificationservice.Config
	JWTSigner           jwtsigner.Config
	OAuth               oauthusecase.Config
}

type RedisConfig struct {
	URL string
}

func (c RedisConfig) Validate() error {
	if c.URL == "" || strings.TrimSpace(c.URL) != c.URL {
		return fmt.Errorf("Redis URL is required")
	}
	return nil
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
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("validate redis config: %w", err)
	}
	if err := c.KafkaProducer.Validate(); err != nil {
		return fmt.Errorf("validate kafka producer config: %w", err)
	}
	if strings.TrimSpace(c.IdentityTopic) == "" {
		return fmt.Errorf("identity topic is required")
	}
	if err := c.Outbox.Validate(); err != nil {
		return fmt.Errorf("validate outbox config: %w", err)
	}
	if c.AuthenticationRedirectURI == "" || strings.TrimSpace(c.AuthenticationRedirectURI) != c.AuthenticationRedirectURI {
		return fmt.Errorf("authentication redirect URI is required")
	}
	if err := c.AccessToken.Validate(); err != nil {
		return fmt.Errorf("validate access token config: %w", err)
	}
	if err := c.IDToken.Validate(); err != nil {
		return fmt.Errorf("validate ID token config: %w", err)
	}
	if err := c.ContactVerification.Validate(); err != nil {
		return fmt.Errorf("validate contact verification config: %w", err)
	}
	if err := c.JWTSigner.Validate(); err != nil {
		return fmt.Errorf("validate JWT signer config: %w", err)
	}
	if err := c.OAuth.Validate(); err != nil {
		return fmt.Errorf("validate OAuth config: %w", err)
	}

	return nil
}
