package interceptor

import (
	"context"
	"log/slog"
	"time"

	"github.com/krackl1n/careeria/grpc/matcher"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func UnaryLogging(logger *slog.Logger) grpc.UnaryServerInterceptor {
	logger = loggerOrDefault(logger)

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		startedAt := time.Now()
		panicked := true

		defer func() {
			code := status.Code(err)
			if panicked {
				code = codes.Internal
			}

			logRPC(
				ctx,
				logger,
				info.FullMethod,
				"unary",
				code,
				time.Since(startedAt),
				err,
			)
		}()

		resp, err = handler(ctx, req)
		panicked = false

		return resp, err
	}
}

func StreamLogging(logger *slog.Logger) grpc.StreamServerInterceptor {
	logger = loggerOrDefault(logger)

	return func(
		srv any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		startedAt := time.Now()
		panicked := true

		defer func() {
			code := status.Code(err)
			if panicked {
				code = codes.Internal
			}

			logRPC(
				stream.Context(),
				logger,
				info.FullMethod,
				streamType(info),
				code,
				time.Since(startedAt),
				err,
			)
		}()

		err = handler(srv, stream)
		panicked = false

		return err
	}
}

func logRPC(
	ctx context.Context,
	logger *slog.Logger,
	fullMethod string,
	rpcType string,
	code codes.Code,
	duration time.Duration,
	err error,
) {
	service, method := matcher.SplitFullMethod(fullMethod)

	args := []any{
		"grpc.service", service,
		"grpc.method", method,
		"grpc.full_method", fullMethod,
		"grpc.type", rpcType,
		"grpc.code", code.String(),
		"duration", duration.String(),
		"request_id", requestIDForLog(ctx),
		"peer_address", peerAddress(ctx),
	}

	if err != nil {
		args = append(args, slog.Any("error", err))
	}

	switch code {
	case codes.OK:
		logger.InfoContext(ctx, "grpc request completed", args...)

	case codes.Canceled,
		codes.InvalidArgument,
		codes.NotFound,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.FailedPrecondition,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unauthenticated:
		logger.WarnContext(ctx, "grpc request completed", args...)

	default:
		logger.ErrorContext(ctx, "grpc request completed", args...)
	}
}

func peerAddress(ctx context.Context) string {
	value, ok := peer.FromContext(ctx)
	if !ok || value.Addr == nil {
		return ""
	}

	return value.Addr.String()
}

func streamType(info *grpc.StreamServerInfo) string {
	switch {
	case info.IsClientStream && info.IsServerStream:
		return "bidi_stream"
	case info.IsClientStream:
		return "client_stream"
	case info.IsServerStream:
		return "server_stream"
	default:
		return "stream"
	}
}
