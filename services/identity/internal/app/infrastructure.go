package app

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	sharedclock "github.com/krackl1n/careeria/clock"
	grpcinterceptor "github.com/krackl1n/careeria/grpc/interceptor"
	sharedidgen "github.com/krackl1n/careeria/idgen"
	sharedmetrics "github.com/krackl1n/careeria/metrics"
	transportgrpc "github.com/krackl1n/careeria/services/identity/internal/transport/grpc"
	"github.com/krackl1n/careeria/txmanager"
)

func (a *App) TransactionManager() *txmanager.Manager[pgx.Tx] {
	if a.transactionManager == nil {
		a.transactionManager = txmanager.New(a.dependencies.PostgresClient)
	}
	return a.transactionManager
}
func (a *App) Clock() sharedclock.Clock {
	if a.clock == nil {
		a.clock = sharedclock.New()
	}
	return a.clock
}
func (a *App) IDGenerator() *sharedidgen.UUID {
	if a.idGenerator == nil {
		generator := sharedidgen.NewUUID()
		a.idGenerator = &generator
	}
	return a.idGenerator
}
func (a *App) ErrorResolver() *transportgrpc.ErrorResolver {
	if a.errorResolver == nil {
		a.errorResolver = transportgrpc.NewErrorResolver()
	}
	return a.errorResolver
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
	return metrics, nil
}
func (a *App) GRPCMetrics() (*grpcinterceptor.Metrics, error) {
	if a.grpcMetrics != nil {
		return a.grpcMetrics, nil
	}
	metrics, err := a.Metrics()
	if err != nil {
		return nil, err
	}
	result, err := grpcinterceptor.NewMetrics(grpcinterceptor.MetricsConfig{Namespace: a.dependencies.Config.Metrics.Namespace, Subsystem: "identity"}, metrics.Registry())
	if err != nil {
		return nil, fmt.Errorf("initialize gRPC metrics: %w", err)
	}
	a.grpcMetrics = result
	return result, nil
}
