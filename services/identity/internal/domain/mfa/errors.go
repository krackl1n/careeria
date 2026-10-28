package mfa

import "errors"

var (
	ErrFactorNotFound           = errors.New("MFA factor not found")
	ErrFactorAlreadyExists      = errors.New("MFA factor already exists")
	ErrEnabledTOTPAlreadyExists = errors.New("an enabled TOTP factor already exists")
	ErrCredentialAlreadyExists  = errors.New("WebAuthn credential already exists")
	ErrFactorDisabled           = errors.New("MFA factor is disabled")
	ErrFactorNotConfirmed       = errors.New("MFA factor is not confirmed")
	ErrTOTPReplay               = errors.New("TOTP time step was already used")
	ErrWebAuthnCounterRollback  = errors.New("WebAuthn signature counter did not increase")
)
