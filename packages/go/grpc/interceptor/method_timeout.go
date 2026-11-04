package interceptor

import (
	"context"
	"time"

	"github.com/krackl1n/careeria/grpc/matcher"
	"google.golang.org/grpc"
)

type TimeoutProvider interface {
	Timeout(fullMethod string) time.Duration
}

type MethodTimeoutConfig struct {
	Provider TimeoutProvider
	Matcher  matcher.Matcher
}

func UnaryMethodTimeout(cfg MethodTimeoutConfig) grpc.UnaryServerInterceptor {
	if cfg.Provider == nil {
		panic("grpc method timeout interceptor: nil provider")
	}

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if !matcher.Matches(cfg.Matcher, info.FullMethod) {
			return handler(ctx, req)
		}

		timeout := cfg.Provider.Timeout(info.FullMethod)
		if timeout <= 0 {
			return handler(ctx, req)
		}

		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		return handler(ctx, req)
	}
}

func StreamMethodTimeout(cfg MethodTimeoutConfig) grpc.StreamServerInterceptor {
	if cfg.Provider == nil {
		panic("grpc method timeout interceptor: nil provider")
	}

	return func(
		srv any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if !matcher.Matches(cfg.Matcher, info.FullMethod) {
			return handler(srv, stream)
		}

		timeout := cfg.Provider.Timeout(info.FullMethod)
		if timeout <= 0 {
			return handler(srv, stream)
		}

		ctx, cancel := context.WithTimeout(stream.Context(), timeout)
		defer cancel()

		return handler(srv, wrapServerStream(stream, ctx))
	}
}
