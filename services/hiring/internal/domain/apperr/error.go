package apperr

import (
	"errors"
	"fmt"
	"strings"
)

type Error struct {
	errorType Type
	code      string
	message   string
	cause     error
}

func New(errorType Type, code, message string, cause error) *Error {
	return &Error{
		errorType: normalizeType(errorType),
		code:      normalizeCode(code),
		message:   normalizeMessage(message),
		cause:     cause,
	}
}

func InvalidArgument(code, message string, cause error) *Error {
	return New(TypeInvalidArgument, code, message, cause)
}

func NotFound(code, message string, cause error) *Error {
	return New(TypeNotFound, code, message, cause)
}

func Conflict(code, message string, cause error) *Error {
	return New(TypeConflict, code, message, cause)
}

func Forbidden(code, message string, cause error) *Error {
	return New(TypeForbidden, code, message, cause)
}

func Unauthorized(code, message string, cause error) *Error {
	return New(TypeUnauthorized, code, message, cause)
}

func Internal(code, message string, cause error) *Error {
	return New(TypeInternal, code, message, cause)
}

func Unavailable(code, message string, cause error) *Error {
	return New(TypeUnavailable, code, message, cause)
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}

	switch {
	case e.cause != nil && e.code != "":
		return fmt.Sprintf("%s: %s: %v", e.code, e.message, e.cause)
	case e.cause != nil:
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	case e.code != "":
		return fmt.Sprintf("%s: %s", e.code, e.message)
	default:
		return e.message
	}
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.cause
}

func (e *Error) Type() Type {
	if e == nil {
		return TypeUnknown
	}

	return e.errorType
}

func (e *Error) Code() string {
	if e == nil {
		return ""
	}

	return e.code
}

func (e *Error) Message() string {
	if e == nil {
		return ""
	}

	return e.message
}

func (e *Error) Is(target error) bool {
	if e == nil || target == nil {
		return e == nil && target == nil
	}

	var appErr *Error
	if !errors.As(target, &appErr) {
		return false
	}

	if appErr.errorType != TypeUnknown && e.errorType != appErr.errorType {
		return false
	}

	if appErr.code != "" && e.code != appErr.code {
		return false
	}

	return true
}

func normalizeType(t Type) Type {
	if !t.Valid() {
		return TypeUnknown
	}
	return t
}

func normalizeCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

func normalizeMessage(message string) string {
	return strings.TrimSpace(message)
}

