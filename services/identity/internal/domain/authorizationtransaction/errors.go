package authorizationtransaction

import "errors"

var (
	ErrAlreadyExists    = errors.New("authorization transaction already exists")
	ErrNotFound         = errors.New("authorization transaction not found")
	ErrExpired          = errors.New("authorization transaction is expired")
	ErrInvalidStatus    = errors.New("authorization transaction status does not allow the operation")
	ErrInvalidStep      = errors.New("authorization transaction step does not allow the operation")
	ErrIdentityMismatch = errors.New("authorization transaction belongs to another identity")
	ErrPKCERequired     = errors.New("PKCE is required")
)
