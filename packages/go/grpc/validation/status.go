package validation

import (
	"errors"
	"strings"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	requestValidationFailed = "request validation failed"
	invalidFieldCode        = "invalid"
)

type allErrors interface {
	AllErrors() []error
}

// fieldError is implemented by generated protoc-gen-validate errors.
// It stays structural so the interceptor package does not depend on a code generator.
type fieldError interface {
	Field() string
	Reason() string
}

// StatusError converts a generated request validation error to an InvalidArgument
// status. Recognized field errors are attached as google.rpc.BadRequest details.
func StatusError(err error) error {
	violations := fieldViolations(err)
	if len(violations) == 0 {
		return status.Error(codes.InvalidArgument, requestValidationFailed)
	}

	grpcStatus, detailsErr := status.New(
		codes.InvalidArgument,
		requestValidationFailed,
	).WithDetails(&errdetails.BadRequest{FieldViolations: violations})
	if detailsErr != nil {
		return status.Error(codes.InvalidArgument, requestValidationFailed)
	}

	return grpcStatus.Err()
}

func fieldViolations(err error) []*errdetails.BadRequest_FieldViolation {
	var result []*errdetails.BadRequest_FieldViolation
	appendFieldViolations(err, &result)

	return result
}

func appendFieldViolations(
	err error,
	target *[]*errdetails.BadRequest_FieldViolation,
) {
	if err == nil {
		return
	}

	var multiple allErrors
	if errors.As(err, &multiple) {
		for _, item := range multiple.AllErrors() {
			appendFieldViolations(item, target)
		}
		return
	}

	var field fieldError
	if !errors.As(err, &field) {
		return
	}

	fieldName := strings.TrimSpace(field.Field())
	if fieldName == "" {
		return
	}

	message := strings.TrimSpace(field.Reason())
	if message == "" {
		message = "field is invalid"
	}

	*target = append(*target, &errdetails.BadRequest_FieldViolation{
		Field:       fieldName,
		Reason:      invalidFieldCode,
		Description: message,
	})
}
