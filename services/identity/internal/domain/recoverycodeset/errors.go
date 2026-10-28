package recoverycodeset

import "errors"

var (
	ErrActiveSetAlreadyExists = errors.New("active recovery code set already exists")
	ErrNotFound               = errors.New("recovery code set not found")
	ErrDisabled               = errors.New("recovery code set is disabled")
	ErrExpired                = errors.New("recovery code set is expired")
	ErrCodeNotFound           = errors.New("recovery code not found")
	ErrCodeUsed               = errors.New("recovery code was already used")
)
