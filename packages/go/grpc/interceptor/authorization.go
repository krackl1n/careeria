package interceptor

import (
	"context"

	"github.com/krackl1n/careeria/grpc/matcher"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Authorizer interface {
	Authorize(ctx context.Context, fullMethod string) error
}

type AuthorizationConfig struct {
	Authorizer Authorizer
	Matcher    matcher.Matcher
}

func UnaryAuthorization(cfg AuthorizationConfig) grpc.UnaryServerInterceptor {
	if cfg.Authorizer == nil {
		panic("grpc authorization interceptor: nil authorizer")
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

		if err := cfg.Authorizer.Authorize(ctx, info.FullMethod); err != nil {
			return nil, normalizeAuthorizationError(err)
		}

		return handler(ctx, req)
	}
}

func StreamAuthorization(cfg AuthorizationConfig) grpc.StreamServerInterceptor {
	if cfg.Authorizer == nil {
		panic("grpc authorization interceptor: nil authorizer")
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

		if err := cfg.Authorizer.Authorize(stream.Context(), info.FullMethod); err != nil {
			return normalizeAuthorizationError(err)
		}

		return handler(srv, stream)
	}
}

func normalizeAuthorizationError(err error) error {
	if _, ok := status.FromError(err); ok {
		return err
	}

	return status.Error(codes.PermissionDenied, "permission denied")
}
