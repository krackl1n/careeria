package interceptor

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PanicHandler interface {
	Handle(ctx context.Context, fullMethod string, recovered any) error
}

func UnaryRecovery(logger *slog.Logger, panicHandler PanicHandler) grpc.UnaryServerInterceptor {
	logger = loggerOrDefault(logger)

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			logger.ErrorContext(
				ctx,
				"grpc panic recovered",
				"grpc.method", info.FullMethod,
				"request_id", requestIDForLog(ctx),
				"panic", fmt.Sprint(recovered),
				"stack", string(debug.Stack()),
			)

			if panicHandler != nil {
				mapped := panicHandler.Handle(ctx, info.FullMethod, recovered)
				if mapped != nil {
					err = mapped
					return
				}
			}

			err = status.Error(codes.Internal, "internal server error")
		}()

		return handler(ctx, req)
	}
}

func StreamRecovery(logger *slog.Logger, panicHandler PanicHandler) grpc.StreamServerInterceptor {
	logger = loggerOrDefault(logger)

	return func(
		srv any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			ctx := stream.Context()

			logger.ErrorContext(
				ctx,
				"grpc panic recovered",
				"grpc.method", info.FullMethod,
				"request_id", requestIDForLog(ctx),
				"panic", fmt.Sprint(recovered),
				"stack", string(debug.Stack()),
			)

			if panicHandler != nil {
				mapped := panicHandler.Handle(ctx, info.FullMethod, recovered)
				if mapped != nil {
					err = mapped
					return
				}
			}

			err = status.Error(codes.Internal, "internal server error")
		}()

		return handler(srv, stream)
	}
}

func loggerOrDefault(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return slog.Default()
	}

	return logger
}
