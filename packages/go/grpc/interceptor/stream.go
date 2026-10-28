package interceptor

import (
	"context"

	"google.golang.org/grpc"
)

type serverStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *serverStream) Context() context.Context {
	return s.ctx
}

func wrapServerStream(stream grpc.ServerStream, ctx context.Context) grpc.ServerStream {
	return &serverStream{
		ServerStream: stream,
		ctx:          ctx,
	}
}
