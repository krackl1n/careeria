package session

import "errors"

var (
	ErrNotFound        = errors.New("session not found")
	ErrRevoked         = errors.New("session is revoked")
	ErrIdleExpired     = errors.New("session idle lifetime has expired")
	ErrAbsoluteExpired = errors.New("session absolute lifetime has expired")
)
