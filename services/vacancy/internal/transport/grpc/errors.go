package grpc

import (
	"errors"

	"github.com/krackl1n/careeria/grpc/interceptor"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
	command "github.com/krackl1n/careeria/services/vacancy/internal/usecase/vacancy"
	query "github.com/krackl1n/careeria/services/vacancy/internal/usecase/vacancyquery"
	"google.golang.org/grpc/codes"
)

type ErrorResolver struct{}

func (ErrorResolver) Resolve(err error) (interceptor.ResolvedError, bool) {
	var code codes.Code
	switch {
	case errors.Is(err, command.ErrInvalidInput), errors.Is(err, query.ErrInvalidInput), errors.Is(err, domain.ErrInvalidDetails), errors.Is(err, domain.ErrInvalidVacancy):
		code = codes.InvalidArgument
	case errors.Is(err, domain.ErrUnauthenticated):
		code = codes.Unauthenticated
	case errors.Is(err, domain.ErrPermissionDenied):
		return interceptor.ResolvedError{Code: codes.PermissionDenied, Message: "permission denied"}, true
	case errors.Is(err, domain.ErrAuthorizationUnavailable):
		return interceptor.ResolvedError{Code: codes.Unavailable, Message: "authorization service unavailable"}, true
	case errors.Is(err, domain.ErrNotFound):
		code = codes.NotFound
	case errors.Is(err, domain.ErrVersionConflict):
		code = codes.Aborted
	case errors.Is(err, command.ErrIdempotencyConflict), errors.Is(err, domain.ErrConflict):
		code = codes.AlreadyExists
	case errors.Is(err, domain.ErrInvalidTransition), errors.Is(err, domain.ErrNotPublishable):
		code = codes.FailedPrecondition
	default:
		return interceptor.ResolvedError{}, false
	}
	return interceptor.ResolvedError{Code: code, Message: err.Error()}, true
}
