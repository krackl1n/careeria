package signingkey

import "errors"

var (
	ErrNotFound                = errors.New("signing key not found")
	ErrInvalidStatus           = errors.New("signing key status does not allow the operation")
	ErrNotYetValid             = errors.New("signing key is not valid yet")
	ErrExpired                 = errors.New("signing key is expired")
	ErrCompromised             = errors.New("signing key is compromised")
	ErrKeyIDAlreadyExists      = errors.New("signing key ID already exists")
	ErrActiveAlgorithmConflict = errors.New("an active signing key already exists for the algorithm")
)
