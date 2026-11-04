package passwordresetchallenge

import "errors"

var (
	ErrNotFound          = errors.New("password reset challenge not found")
	ErrExpired           = errors.New("password reset challenge is expired")
	ErrConsumed          = errors.New("password reset challenge is consumed")
	ErrRevoked           = errors.New("password reset challenge is revoked")
	ErrAttemptsExhausted = errors.New("password reset challenge attempts are exhausted")
)
