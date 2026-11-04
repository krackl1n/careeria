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

func UnaryValidation() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if err := validateMessage(req); err != nil {
			return validation.StatusError(err)
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func StreamValidation() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return nil, err
		}

		return &validatingClientStream{
			ClientStream: stream,
		}, nil
	}
}

type validatingClientStream struct {
	grpc.ClientStream
}

func (s *validatingClientStream) SendMsg(message any) error {
	if err := validateMessage(message); err != nil {
		return validation.StatusError(err)
	}

	return s.ClientStream.SendMsg(message)
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
