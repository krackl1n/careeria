package client

import (
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

func buildDialOptions(cfg Config, extraOptions ...grpc.DialOption) ([]grpc.DialOption, error) {
	options := make([]grpc.DialOption, 0, len(extraOptions)+10)

	transportOption, err := buildTransportOption(cfg)
	if err != nil {
		return nil, err
	}

	options = append(options, transportOption)

	if cfg.UserAgent != "" {
		options = append(options, grpc.WithUserAgent(cfg.UserAgent))
	}

	callOptions := buildCallOptions(cfg)
	if len(callOptions) > 0 {
		options = append(options, grpc.WithDefaultCallOptions(callOptions...))
	}

	if cfg.Keepalive.Enabled {
		options = append(options, buildKeepaliveOption(cfg.Keepalive))
	}

	if cfg.Tracing.Enabled {
		options = append(options, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	}

	options = append(options, extraOptions...)

	return options, nil
}

func buildTransportOption(cfg Config) (grpc.DialOption, error) {
	if cfg.Insecure {
		return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
	}

	creds, err := newTransportCredentials(cfg.TLS)
	if err != nil {
		return nil, fmt.Errorf("create transport credentials: %w", err)
	}

	return grpc.WithTransportCredentials(creds), nil
}

func buildCallOptions(cfg Config) []grpc.CallOption {
	options := make([]grpc.CallOption, 0, 2)

	if cfg.MaxRecvMessageSize > 0 {
		options = append(options, grpc.MaxCallRecvMsgSize(cfg.MaxRecvMessageSize))
	}

	if cfg.MaxSendMessageSize > 0 {
		options = append(options, grpc.MaxCallSendMsgSize(cfg.MaxSendMessageSize))
	}

	return options
}

func buildKeepaliveOption(cfg KeepaliveConfig) grpc.DialOption {
	return grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                durationOrDefault(cfg.Time, 30*time.Second),
		Timeout:             durationOrDefault(cfg.Timeout, 10*time.Second),
		PermitWithoutStream: cfg.PermitWithoutStream,
	})
}

func durationOrDefault(value time.Duration, fallback time.Duration) time.Duration {
	if value > 0 {
		return value
	}

	return fallback
}
