package refreshtokenfamily

import "errors"

var (
	ErrSessionFamilyAlreadyExists = errors.New("refresh token family already exists for session")
	ErrNotFound                   = errors.New("refresh token family not found")
	ErrRevoked                    = errors.New("refresh token family is revoked")
	ErrExpired                    = errors.New("refresh token family is expired")
	ErrTokenNotFound              = errors.New("refresh token not found")
	ErrTokenRevoked               = errors.New("refresh token is revoked")
	ErrTokenExpired               = errors.New("refresh token is expired")
)
