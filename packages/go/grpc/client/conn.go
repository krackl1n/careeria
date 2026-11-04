package client

import (
	"fmt"

	"google.golang.org/grpc"
)

func New(cfg Config, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate grpc config: %w", err)
	}

	dialOptions, err := buildDialOptions(cfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("build grpc dial options: %w", err)
	}

	conn, err := grpc.NewClient(cfg.Target, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("create grpc client connection: %w", err)
	}

	return conn, nil
}
