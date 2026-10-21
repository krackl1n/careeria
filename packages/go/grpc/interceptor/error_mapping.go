package interceptor

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"
)

type ResolvedError struct {
	Code    codes.Code
	Message string
	Details []proto.Message
}

type ErrorResolver interface {
	Resolve(error) (ResolvedError, bool)
}

func UnaryErrorMapping(
	logger *slog.Logger,
	resolver ErrorResolver,
) grpc.UnaryServerInterceptor {
	logger = loggerOrDefault(logger)

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}

		return resp, resolveError(ctx, logger, info.FullMethod, resolver, err)
	}
}

func StreamErrorMapping(
	logger *slog.Logger,
	resolver ErrorResolver,
) grpc.StreamServerInterceptor {
	logger = loggerOrDefault(logger)

	return func(
		srv any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		err := handler(srv, stream)
		if err == nil {
			return nil
		}

		return resolveError(stream.Context(), logger, info.FullMethod, resolver, err)
	}
}

func resolveError(
	ctx context.Context,
	logger *slog.Logger,
	fullMethod string,
	resolver ErrorResolver,
	err error,
) error {
	if resolved, ok := preserveStatusError(err); ok {
		return resolved
	}

	if resolver != nil {
		if resolved, ok := resolver.Resolve(err); ok {
			logMappedServerError(ctx, logger, fullMethod, resolved, err)
			return buildStatus(resolved)
		}
	}

	if resolved, ok := resolveContextError(err); ok {
		return buildStatus(resolved)
	}

	logger.ErrorContext(
		ctx,
		"unmapped grpc error",
		"grpc.method", fullMethod,
		"request_id", requestIDForLog(ctx),
		"error", err,
	)

	return status.Error(codes.Internal, "internal server error")
}

func logMappedServerError(
	ctx context.Context,
	logger *slog.Logger,
	fullMethod string,
	resolved ResolvedError,
	err error,
) {
	switch resolved.Code {
	case codes.Internal, codes.Unknown, codes.DataLoss, codes.Unavailable:
		logger.ErrorContext(
			ctx,
			"mapped grpc server error",
			"grpc.method", fullMethod,
			"grpc.code", resolved.Code.String(),
			"request_id", requestIDForLog(ctx),
			"error", err,
		)
	}
}

func preserveStatusError(err error) (error, bool) {
	value, ok := status.FromError(err)
	if !ok {
		return nil, false
	}

	resolved := sanitizeResolvedError(ResolvedError{
		Code:    value.Code(),
		Message: value.Message(),
	})
	if resolved.Code == value.Code() && resolved.Message == value.Message() {
		return err, true
	}

	return status.Error(resolved.Code, resolved.Message), true
}

func resolveContextError(err error) (ResolvedError, bool) {
	switch {
	case errors.Is(err, context.Canceled):
		return ResolvedError{
			Code:    codes.Canceled,
			Message: "request canceled",
		}, true

	case errors.Is(err, context.DeadlineExceeded):
		return ResolvedError{
			Code:    codes.DeadlineExceeded,
			Message: "deadline exceeded",
		}, true

	default:
		return ResolvedError{}, false
	}
}

func buildStatus(resolved ResolvedError) error {
	resolved = sanitizeResolvedError(resolved)
	value := status.New(resolved.Code, resolved.Message)
	if len(resolved.Details) == 0 {
		return value.Err()
	}

	details := make([]protoadapt.MessageV1, len(resolved.Details))
	for index := range resolved.Details {
		details[index] = protoadapt.MessageV1Of(resolved.Details[index])
	}
	withDetails, err := value.WithDetails(details...)
	if err != nil {
		return value.Err()
	}

	return withDetails.Err()
}

func sanitizeResolvedError(resolved ResolvedError) ResolvedError {
	switch resolved.Code {
	case codes.OK, codes.Unknown, codes.Internal:
		return internalResolvedError()

	case codes.DataLoss:
		return ResolvedError{
			Code:    codes.DataLoss,
			Message: "data loss",
		}

	case codes.Unavailable:
		return ResolvedError{
			Code:    codes.Unavailable,
			Message: "service unavailable",
		}

	default:
		if resolved.Message == "" {
			resolved.Message = defaultCodeMessage(resolved.Code)
		}

		return resolved
	}
}

func internalResolvedError() ResolvedError {
	return ResolvedError{
		Code:    codes.Internal,
		Message: "internal server error",
	}
}

func defaultCodeMessage(code codes.Code) string {
	switch code {
	case codes.Canceled:
		return "request canceled"
	case codes.InvalidArgument:
		return "invalid argument"
	case codes.DeadlineExceeded:
		return "deadline exceeded"
	case codes.NotFound:
		return "not found"
	case codes.AlreadyExists:
		return "already exists"
	case codes.PermissionDenied:
		return "permission denied"
	case codes.ResourceExhausted:
		return "resource exhausted"
	case codes.FailedPrecondition:
		return "failed precondition"
	case codes.Aborted:
		return "request aborted"
	case codes.OutOfRange:
		return "out of range"
	case codes.Unimplemented:
		return "not implemented"
	case codes.Unavailable:
		return "service unavailable"
	case codes.Unauthenticated:
		return "unauthenticated"
	default:
		return "internal server error"
	}
}
