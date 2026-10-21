package interceptor

import (
	"context"

	"github.com/krackl1n/careeria/grpc/validation"
	"google.golang.org/grpc"
)

type validateAll interface {
	ValidateAll() error
}

type validate interface {
	Validate() error
}

func UnaryValidation() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if err := validateMessage(req); err != nil {
			return nil, validation.StatusError(err)
		}

		return handler(ctx, req)
	}
}

func StreamValidation() grpc.StreamServerInterceptor {
	return func(
		srv any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		return handler(srv, &validatingServerStream{ServerStream: stream})
	}
}

type validatingServerStream struct {
	grpc.ServerStream
}

func (s *validatingServerStream) RecvMsg(message any) error {
	if err := s.ServerStream.RecvMsg(message); err != nil {
		return err
	}

	if err := validateMessage(message); err != nil {
		return validation.StatusError(err)
	}

	return nil
}

func validateMessage(message any) error {
	switch value := message.(type) {
	case validateAll:
		return value.ValidateAll()
	case validate:
		return value.Validate()
	default:
		return nil
	}
}
