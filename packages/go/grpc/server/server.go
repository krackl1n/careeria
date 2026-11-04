package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	cfg Config

	grpc   *grpc.Server
	health *health.Server
}

func (s *Server) Name() string {
	return "grpc"
}

func New(cfg Config, opts ...Option) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate grpc config: %w", err)
	}

	var o options

	for _, opt := range opts {
		opt(&o)
	}

	serverOptions := make([]grpc.ServerOption, 0, 8)

	if cfg.ConnectionTimeout > 0 {
		serverOptions = append(serverOptions,
			grpc.ConnectionTimeout(cfg.ConnectionTimeout),
		)
	}

	if cfg.MaxRecvMsgSize > 0 {
		serverOptions = append(serverOptions,
			grpc.MaxRecvMsgSize(cfg.MaxRecvMsgSize),
		)
	}

	if cfg.MaxSendMsgSize > 0 {
		serverOptions = append(serverOptions,
			grpc.MaxSendMsgSize(cfg.MaxSendMsgSize),
		)
	}

	if len(o.unaryInterceptors) > 0 {
		serverOptions = append(serverOptions,
			grpc.ChainUnaryInterceptor(o.unaryInterceptors...),
		)
	}

	if len(o.streamInterceptors) > 0 {
		serverOptions = append(serverOptions,
			grpc.ChainStreamInterceptor(o.streamInterceptors...),
		)
	}

	serverOptions = append(serverOptions, o.serverOptions...)

	grpcServer := grpc.NewServer(serverOptions...)

	var healthServer *health.Server

	if cfg.EnableHealth {
		healthServer = health.NewServer()

		healthpb.RegisterHealthServer(
			grpcServer,
			healthServer,
		)

		healthServer.SetServingStatus(
			"",
			healthpb.HealthCheckResponse_SERVING,
		)
	}

	if cfg.EnableReflection {
		reflection.Register(grpcServer)
	}

	return &Server{
		cfg:    cfg,
		grpc:   grpcServer,
		health: healthServer,
	}, nil
}

func (s *Server) GRPC() *grpc.Server {
	return s.grpc
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}

	if err := s.grpc.Serve(listener); err != nil &&
		!errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("serve grpc: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.health != nil {
		s.health.SetServingStatus(
			"",
			healthpb.HealthCheckResponse_NOT_SERVING,
		)
	}

	done := make(chan struct{})

	go func() {
		s.grpc.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil

	case <-ctx.Done():
		s.grpc.Stop()
		return ctx.Err()
	}
}
