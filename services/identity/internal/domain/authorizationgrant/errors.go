package authorizationgrant

import "errors"

var (
	ErrAlreadyExists = errors.New("authorization grant already exists")
	ErrNotFound      = errors.New("authorization grant not found")
	ErrRevoked       = errors.New("authorization grant is revoked")
	ErrExpired       = errors.New("authorization grant is expired")
)
