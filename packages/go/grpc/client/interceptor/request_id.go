package interceptor

import (
	"context"

	"github.com/krackl1n/careeria/ctxmeta"
	"google.golang.org/grpc"
)

const (
	RequestIDMetadataKey     = "x-request-id"
	CorrelationIDMetadataKey = "x-correlation-id"
)

func UnaryRequestID() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx = withRequestID(ctx)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func StreamRequestID() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		ctx = withRequestID(ctx)
		return streamer(ctx, desc, cc, method, opts...)
	}
}

func withRequestID(ctx context.Context) context.Context {
	if metadataValue(ctx, RequestIDMetadataKey) == "" {
		if requestID, ok := ctxmeta.RequestID(ctx); ok {
			ctx = withMetadataValue(
				ctx,
				RequestIDMetadataKey,
				requestID,
				false,
			)
		}
	}

	if metadataValue(ctx, CorrelationIDMetadataKey) == "" {
		if correlationID, ok := ctxmeta.CorrelationID(ctx); ok {
			ctx = withMetadataValue(
				ctx,
				CorrelationIDMetadataKey,
				correlationID,
				false,
			)
		}
	}

	return ctx
}
