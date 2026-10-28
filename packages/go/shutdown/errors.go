package shutdown

import "errors"

var (
	ErrNilContext     = errors.New("shutdown: context is required")
	ErrInvalidTimeout = errors.New("shutdown: timeout must be positive")
	ErrNilComponent   = errors.New("shutdown: component is required")
)
