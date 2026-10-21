package preset

import (
	"log/slog"

	"github.com/krackl1n/careeria/grpc/interceptor"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

type ServerConfig struct {
	Logger *slog.Logger

	RequestID interceptor.RequestIDConfig
	Metrics   *interceptor.Metrics

	ErrorResolver interceptor.ErrorResolver

	Authentication *interceptor.AuthenticationConfig
	TenantContext  *interceptor.TenantContextConfig
	Authorization  *interceptor.AuthorizationConfig
	MethodTimeout  *interceptor.MethodTimeoutConfig
	Idempotency    *interceptor.IdempotencyConfig

	PanicHandler interceptor.PanicHandler

	OpenTelemetry []otelgrpc.Option
}

func Server(cfg ServerConfig) ServerPreset {
	if cfg.Metrics == nil {
		panic("grpc server preset: nil metrics")
	}

	unary := []grpc.UnaryServerInterceptor{
		interceptor.UnaryRecovery(cfg.Logger, cfg.PanicHandler),
		interceptor.UnaryRequestID(cfg.RequestID),
		interceptor.UnaryLogging(cfg.Logger),
		cfg.Metrics.UnaryServerInterceptor(),
		interceptor.UnaryErrorMapping(cfg.Logger, cfg.ErrorResolver),
	}

	stream := []grpc.StreamServerInterceptor{
		interceptor.StreamRecovery(cfg.Logger, cfg.PanicHandler),
		interceptor.StreamRequestID(cfg.RequestID),
		interceptor.StreamLogging(cfg.Logger),
		cfg.Metrics.StreamServerInterceptor(),
		interceptor.StreamErrorMapping(cfg.Logger, cfg.ErrorResolver),
	}

	if cfg.Authentication != nil {
		unary = append(unary, interceptor.UnaryAuthentication(*cfg.Authentication))
		stream = append(stream, interceptor.StreamAuthentication(*cfg.Authentication))
	}

	if cfg.TenantContext != nil {
		unary = append(unary, interceptor.UnaryTenantContext(*cfg.TenantContext))
		stream = append(stream, interceptor.StreamTenantContext(*cfg.TenantContext))
	}

	if cfg.Authorization != nil {
		unary = append(unary, interceptor.UnaryAuthorization(*cfg.Authorization))
		stream = append(stream, interceptor.StreamAuthorization(*cfg.Authorization))
	}

	if cfg.MethodTimeout != nil {
		unary = append(unary, interceptor.UnaryMethodTimeout(*cfg.MethodTimeout))
		stream = append(stream, interceptor.StreamMethodTimeout(*cfg.MethodTimeout))
	}

	unary = append(unary, interceptor.UnaryValidation())
	stream = append(stream, interceptor.StreamValidation())

	if cfg.Idempotency != nil {
		unary = append(unary, interceptor.UnaryIdempotency(*cfg.Idempotency))
	}

	return ServerPreset{
		Unary:        unary,
		Stream:       stream,
		StatsHandler: otelgrpc.NewServerHandler(cfg.OpenTelemetry...),
	}
}
