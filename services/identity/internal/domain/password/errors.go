package password

import "errors"

var (
	ErrAlreadyExists = errors.New("password already exists")
	ErrNotFound      = errors.New("password not found")
	ErrLocked        = errors.New("password authentication is locked")
	ErrExpired       = errors.New("password is expired")
	ErrCompromised   = errors.New("password is compromised")
)
