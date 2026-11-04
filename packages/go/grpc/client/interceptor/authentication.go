package interceptor

import (
	"context"

	"github.com/krackl1n/careeria/grpc/matcher"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type CredentialsProvider interface {
	Metadata(ctx context.Context, fullMethod string) (metadata.MD, error)
}

type CredentialsProviderFunc func(
	ctx context.Context,
	fullMethod string,
) (metadata.MD, error)

func (f CredentialsProviderFunc) Metadata(
	ctx context.Context,
	fullMethod string,
) (metadata.MD, error) {
	return f(ctx, fullMethod)
}

type AuthenticationDeps struct {
	Provider CredentialsProvider
	Matcher  matcher.Matcher
}

func UnaryAuthentication(deps AuthenticationDeps) grpc.UnaryClientInterceptor {
	if deps.Provider == nil {
		panic("grpc client authentication interceptor: nil credentials provider")
	}

	return func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if !matcher.Matches(deps.Matcher, method) {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		values, err := deps.Provider.Metadata(ctx, method)
		if err != nil {
			return err
		}

		ctx = withMetadata(ctx, values, true)

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func StreamAuthentication(deps AuthenticationDeps) grpc.StreamClientInterceptor {
	if deps.Provider == nil {
		panic("grpc client authentication interceptor: nil credentials provider")
	}

	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		if !matcher.Matches(deps.Matcher, method) {
			return streamer(ctx, desc, cc, method, opts...)
		}

		values, err := deps.Provider.Metadata(ctx, method)
		if err != nil {
			return nil, err
		}

		ctx = withMetadata(ctx, values, true)

		return streamer(ctx, desc, cc, method, opts...)
	}
}
