package client

import (
	"fmt"
	"time"
)

type Config struct {
	Target string

	Insecure bool
	TLS      TLSConfig

	UserAgent string

	MaxRecvMessageSize int
	MaxSendMessageSize int

	Keepalive KeepaliveConfig

	Tracing TracingConfig
}

type TLSConfig struct {
	Enabled bool

	CAFile string

	ClientCertFile string
	ClientKeyFile  string

	ServerName string
}

type KeepaliveConfig struct {
	Enabled bool

	Time                time.Duration
	Timeout             time.Duration
	PermitWithoutStream bool
}

type TracingConfig struct {
	Enabled bool
}

func (c Config) Validate() error {
	if c.Target == "" {
		return fmt.Errorf("grpc target is required")
	}

	if c.Insecure && c.TLS.Enabled {
		return fmt.Errorf("grpc insecure and tls cannot be enabled together")
	}

	if !c.Insecure && !c.TLS.Enabled {
		return fmt.Errorf("grpc tls must be enabled when insecure is false")
	}

	if c.MaxRecvMessageSize < 0 {
		return fmt.Errorf("grpc max recv message size must not be negative")
	}

	if c.MaxSendMessageSize < 0 {
		return fmt.Errorf("grpc max send message size must not be negative")
	}

	if c.Keepalive.Time < 0 {
		return fmt.Errorf("grpc keepalive time must not be negative")
	}

	if c.Keepalive.Timeout < 0 {
		return fmt.Errorf("grpc keepalive timeout must not be negative")
	}

	if c.TLS.Enabled {
		if c.TLS.ClientCertFile != "" && c.TLS.ClientKeyFile == "" {
			return fmt.Errorf("grpc client key file is required")
		}

		if c.TLS.ClientKeyFile != "" && c.TLS.ClientCertFile == "" {
			return fmt.Errorf("grpc client cert file is required")
		}
	}

	return nil
}
