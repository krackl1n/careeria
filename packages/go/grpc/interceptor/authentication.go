package interceptor

import (
	"context"

	"github.com/krackl1n/careeria/ctxmeta"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/krackl1n/careeria/grpc/matcher"
)

type Authenticator interface {
	Authenticate(ctx context.Context, fullMethod string) (ctxmeta.Principal, error)
}

type AuthenticationConfig struct {
	Authenticator Authenticator
	Matcher       matcher.Matcher
}

func UnaryAuthentication(cfg AuthenticationConfig) grpc.UnaryServerInterceptor {
	if cfg.Authenticator == nil {
		panic("grpc authentication interceptor: nil authenticator")
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

		principal, err := cfg.Authenticator.Authenticate(ctx, info.FullMethod)
		if err != nil {
			return nil, normalizeAuthenticationError(err)
		}

		if principal.SubjectID == "" {
			return nil, status.Error(codes.Unauthenticated, "unauthenticated")
		}

		return handler(ctxmeta.WithPrincipal(ctx, principal), req)
	}
}

func StreamAuthentication(cfg AuthenticationConfig) grpc.StreamServerInterceptor {
	if cfg.Authenticator == nil {
		panic("grpc authentication interceptor: nil authenticator")
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

		ctx := stream.Context()

		principal, err := cfg.Authenticator.Authenticate(ctx, info.FullMethod)
		if err != nil {
			return normalizeAuthenticationError(err)
		}

		if principal.SubjectID == "" {
			return status.Error(codes.Unauthenticated, "unauthenticated")
		}

		ctx = ctxmeta.WithPrincipal(ctx, principal)

		return handler(srv, wrapServerStream(stream, ctx))
	}
}

func normalizeAuthenticationError(err error) error {
	if _, ok := status.FromError(err); ok {
		return err
	}

	return status.Error(codes.Unauthenticated, "unauthenticated")
}
