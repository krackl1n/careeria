package emailverificationchallenge

import "errors"

var (
	ErrNotFound          = errors.New("email verification challenge not found")
	ErrExpired           = errors.New("email verification challenge is expired")
	ErrConsumed          = errors.New("email verification challenge is consumed")
	ErrRevoked           = errors.New("email verification challenge is revoked")
	ErrAttemptsExhausted = errors.New("email verification challenge attempts are exhausted")
)
