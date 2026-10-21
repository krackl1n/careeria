package app

import (
	"context"
	"errors"
	"fmt"

	sharedclock "github.com/krackl1n/careeria/clock"
	grpcinterceptor "github.com/krackl1n/careeria/grpc/interceptor"
	grpcpreset "github.com/krackl1n/careeria/grpc/preset"
	grpcserver "github.com/krackl1n/careeria/grpc/server"
	sharedidgen "github.com/krackl1n/careeria/idgen"
	sharedmetrics "github.com/krackl1n/careeria/metrics"
	sharedruntime "github.com/krackl1n/careeria/runtime"
	"google.golang.org/grpc"
)

type App struct {
	dependencies *Dependencies
	clock        sharedclock.Clock
	idGenerator  *sharedidgen.UUID
	metrics      *sharedmetrics.Component
	grpcMetrics  *grpcinterceptor.Metrics
	grpcServer   *grpcserver.Server
}

func New(dependencies *Dependencies) (*App, error) {
	if dependencies == nil {
		return nil, errors.New("application dependencies are required")
	}
	if dependencies.Logger == nil {
		return nil, errors.New("application logger is required")
	}
	if dependencies.Closer == nil {
		return nil, errors.New("application closer is required")
	}

	idGen := sharedidgen.NewUUID()
	return &App{
		dependencies: dependencies,
		clock:        sharedclock.New(),
		idGenerator:  &idGen,
	}, nil
}

func (a *App) Runtimes(ctx context.Context) ([]sharedruntime.Component, error) {
	metrics, err := a.Metrics()
	if err != nil {
		return nil, fmt.Errorf("initialize metrics: %w", err)
	}

	server, err := a.GRPCServer()
	if err != nil {
		return nil, fmt.Errorf("initialize gRPC server: %w", err)
	}

	components := []sharedruntime.Component{server}
	if a.dependencies.Config.Metrics.Enabled {
		components = append(components, metrics)
	}

	return components, nil
}

func (a *App) Metrics() (*sharedmetrics.Component, error) {
	if a == nil || a.dependencies == nil {
		return nil, errors.New("application is not initialized")
	}
	if a.metrics != nil {
		return a.metrics, nil
	}

	metrics, err := sharedmetrics.New(a.dependencies.Config.Metrics)
	if err != nil {
		return nil, err
	}

	a.metrics = metrics
	return a.metrics, nil
}

func (a *App) GRPCMetrics() (*grpcinterceptor.Metrics, error) {
	if a.grpcMetrics != nil {
		return a.grpcMetrics, nil
	}

	metrics, err := a.Metrics()
	if err != nil {
		return nil, err
	}

	result, err := grpcinterceptor.NewMetrics(
		grpcinterceptor.MetricsConfig{
			Namespace: a.dependencies.Config.Metrics.Namespace,
			Subsystem: "hiring",
		},
		metrics.Registry(),
	)
	if err != nil {
		return nil, fmt.Errorf("initialize gRPC metrics: %w", err)
	}

	a.grpcMetrics = result
	return result, nil
}

func (a *App) GRPCServer() (*grpcserver.Server, error) {
	if a.grpcServer != nil {
		return a.grpcServer, nil
	}

	metrics, err := a.GRPCMetrics()
	if err != nil {
		return nil, err
	}

	preset := grpcpreset.Server(grpcpreset.ServerConfig{
		Logger:  a.dependencies.Logger,
		Metrics: metrics,
	})

	server, err := grpcserver.New(
		a.dependencies.Config.GRPCServer,
		grpcserver.WithUnaryInterceptors(preset.Unary...),
		grpcserver.WithStreamInterceptors(preset.Stream...),
		grpcserver.WithServerOptions(grpc.StatsHandler(preset.StatsHandler)),
	)
	if err != nil {
		return nil, err
	}

	a.grpcServer = server
	return a.grpcServer, nil
}

