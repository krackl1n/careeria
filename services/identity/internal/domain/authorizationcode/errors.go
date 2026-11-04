package authorizationcode

import "errors"

var (
	ErrAlreadyExists          = errors.New("authorization code already exists")
	ErrNotFound               = errors.New("authorization code not found")
	ErrExpired                = errors.New("authorization code is expired")
	ErrOAuthClientMismatch    = errors.New("authorization code belongs to another OAuth client")
	ErrRedirectURIMismatch    = errors.New("authorization code redirect URI does not match")
	ErrPKCERequired           = errors.New("PKCE code verifier is required")
	ErrPKCEVerificationFailed = errors.New("PKCE verification failed")
)
