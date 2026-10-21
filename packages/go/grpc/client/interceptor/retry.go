package interceptor

import (
	"context"

	"google.golang.org/grpc"
)

type UnaryRetryFunc func(
	ctx context.Context,
	method string,
	attempt func(context.Context) error,
) error

type StreamRetryFunc func(
	ctx context.Context,
	method string,
	desc *grpc.StreamDesc,
	open func(context.Context) (grpc.ClientStream, error),
) (grpc.ClientStream, error)

func UnaryRetry(retry UnaryRetryFunc) grpc.UnaryClientInterceptor {
	if retry == nil {
		panic("grpc client retry interceptor: nil retry function")
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
		return retry(ctx, method, func(attemptCtx context.Context) error {
			return invoker(attemptCtx, method, req, reply, cc, opts...)
		})
	}
}

func StreamRetry(retry StreamRetryFunc) grpc.StreamClientInterceptor {
	if retry == nil {
		panic("grpc client retry interceptor: nil retry function")
	}

	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		return retry(ctx, method, desc, func(attemptCtx context.Context) (grpc.ClientStream, error) {
			return streamer(attemptCtx, desc, cc, method, opts...)
		})
	}
}
