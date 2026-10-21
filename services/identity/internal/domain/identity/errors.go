package identity

import "errors"

var (
	ErrNotFound                = errors.New("identity not found")
	ErrInvalidStatus           = errors.New("identity status does not allow this operation")
	ErrInvalidTransition       = errors.New("invalid identity status transition")
	ErrLoginMethodRequired     = errors.New("identity must have at least one login method")
	ErrEmailNotFound           = errors.New("identity email not found")
	ErrEmailAlreadyExists      = errors.New("identity email already exists")
	ErrPhoneNotFound           = errors.New("identity phone not found")
	ErrPhoneAlreadyExists      = errors.New("identity phone already exists")
	ErrExternalAccountNotFound = errors.New("external identity account not found")
	ErrExternalAccountExists   = errors.New("external identity account already exists")
	ErrExternalAccountDisabled = errors.New("external identity account is disabled")
)
