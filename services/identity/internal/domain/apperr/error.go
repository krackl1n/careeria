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

func As(err error) (*Error, bool) {
	var appErr *Error

	if !errors.As(err, &appErr) {
		return nil, false
	}

	return appErr, true
}

func IsType(err error, errorType Type) bool {
	appErr, ok := As(err)
	return ok && appErr.Type() == errorType
}

func IsCode(err error, code string) bool {
	appErr, ok := As(err)
	return ok && appErr.Code() == code
}

func normalizeType(errorType Type) Type {
	if !errorType.Valid() {
		return TypeInternal
	}

	return errorType
}

func normalizeCode(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return "internal_error"
	}

	return code
}

func normalizeMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "internal error"
	}

	return message
}
