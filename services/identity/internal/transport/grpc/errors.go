package grpc

import (
	grpcinterceptor "github.com/krackl1n/careeria/grpc/interceptor"
	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

const errorDomain = "careeria.identity"

type ErrorResolver struct{}

func NewErrorResolver() *ErrorResolver {
	return &ErrorResolver{}
}

func (*ErrorResolver) Resolve(err error) (grpcinterceptor.ResolvedError, bool) {
	value, ok := apperr.As(err)
	if !ok {
		return grpcinterceptor.ResolvedError{}, false
	}

	return grpcinterceptor.ResolvedError{
		Code:    grpcCode(value.Type()),
		Message: value.Message(),
		Details: []proto.Message{&errdetails.ErrorInfo{
			Reason: value.Code(),
			Domain: errorDomain,
		}},
	}, true
}

func grpcCode(value apperr.Type) codes.Code {
	switch value {
	case apperr.TypeInvalidArgument:
		return codes.InvalidArgument
	case apperr.TypeUnauthorized:
		return codes.Unauthenticated
	case apperr.TypeForbidden:
		return codes.PermissionDenied
	case apperr.TypeNotFound:
		return codes.NotFound
	case apperr.TypeConflict:
		return codes.FailedPrecondition
	case apperr.TypeUnavailable:
		return codes.Unavailable
	default:
		return codes.Internal
	}
}
