package serviceaccount

import "errors"

var (
	ErrNotFound                 = errors.New("service account not found")
	ErrAccountKeyAlreadyExists  = errors.New("service account key already exists in organization")
	ErrInvalidStatus            = errors.New("service account status does not allow this operation")
	ErrUsableCredentialRequired = errors.New("a usable credential is required")
	ErrCredentialNotFound       = errors.New("service account credential not found")
	ErrCredentialRevoked        = errors.New("service account credential is revoked")
	ErrCredentialExpired        = errors.New("service account credential is expired")
)
