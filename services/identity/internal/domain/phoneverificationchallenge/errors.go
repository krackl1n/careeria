package phoneverificationchallenge

import "errors"

var (
	ErrNotFound          = errors.New("phone verification challenge not found")
	ErrExpired           = errors.New("phone verification challenge is expired")
	ErrConsumed          = errors.New("phone verification challenge is consumed")
	ErrRevoked           = errors.New("phone verification challenge is revoked")
	ErrAttemptsExhausted = errors.New("phone verification challenge attempts are exhausted")
)
