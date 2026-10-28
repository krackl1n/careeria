package interceptor

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/krackl1n/careeria/grpc/matcher"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryLogging(logger *slog.Logger) grpc.UnaryClientInterceptor {
	logger = loggerOrDefault(logger)

	return func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		startedAt := time.Now()

		err := invoker(ctx, method, req, reply, cc, opts...)

		logRPC(
			ctx,
			logger,
			cc.Target(),
			method,
			"unary",
			time.Since(startedAt),
			err,
		)

		return err
	}
}

func StreamLogging(logger *slog.Logger) grpc.StreamClientInterceptor {
	logger = loggerOrDefault(logger)

	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		startedAt := time.Now()

		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			logRPC(
				ctx,
				logger,
				cc.Target(),
				method,
				streamType(desc),
				time.Since(startedAt),
				err,
			)

			return nil, err
		}

		return wrapClientStream(stream, ctx, desc, func(err error) {
			logRPC(
				ctx,
				logger,
				cc.Target(),
				method,
				streamType(desc),
				time.Since(startedAt),
				err,
			)
		}), nil
	}
}

func logRPC(
	ctx context.Context,
	logger *slog.Logger,
	target string,
	fullMethod string,
	rpcType string,
	duration time.Duration,
	err error,
) {
	service, method := matcher.SplitFullMethod(fullMethod)
	code := codeFromError(err)

	args := []any{
		"grpc.target", target,
		"grpc.service", service,
		"grpc.method", method,
		"grpc.full_method", fullMethod,
		"grpc.type", rpcType,
		"grpc.code", code.String(),
		"duration", duration.String(),
		"duration_ms", float64(duration.Microseconds()) / 1000,
		"request_id", metadataValue(ctx, RequestIDMetadataKey),
	}

	if logErr := errorForLog(err); logErr != nil {
		args = append(args, slog.Any("error", logErr))
	}

	switch code {
	case codes.OK:
		logger.InfoContext(ctx, "grpc client request completed", args...)

	case codes.Canceled,
		codes.InvalidArgument,
		codes.DeadlineExceeded,
		codes.NotFound,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.ResourceExhausted,
		codes.FailedPrecondition,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unavailable,
		codes.Unauthenticated:
		logger.WarnContext(ctx, "grpc client request completed", args...)

	default:
		logger.ErrorContext(ctx, "grpc client request completed", args...)
	}
}

func streamType(desc *grpc.StreamDesc) string {
	switch {
	case desc == nil:
		return "stream"
	case desc.ClientStreams && desc.ServerStreams:
		return "bidi_stream"
	case desc.ClientStreams:
		return "client_stream"
	case desc.ServerStreams:
		return "server_stream"
	default:
		return "stream"
	}
}

func codeFromError(err error) codes.Code {
	switch {
	case err == nil:
		return codes.OK
	case errors.Is(err, io.EOF):
		return codes.OK
	case errors.Is(err, context.Canceled):
		return codes.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return codes.DeadlineExceeded
	default:
		return status.Code(err)
	}
}

func errorForLog(err error) error {
	if errors.Is(err, io.EOF) {
		return nil
	}

	return err
}

func loggerOrDefault(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return slog.Default()
	}

	return logger
}
