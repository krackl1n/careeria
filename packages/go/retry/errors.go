package retry

import "errors"

var (
	ErrNilContext   = errors.New("retry: context is nil")
	ErrNilOperation = errors.New("retry: operation is nil")
)
