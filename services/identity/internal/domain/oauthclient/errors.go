package oauthclient

import "errors"

var (
	ErrClientIDAlreadyExists              = errors.New("OAuth client identifier already exists")
	ErrNotFound                           = errors.New("OAuth client not found")
	ErrInvalidStatus                      = errors.New("OAuth client status does not allow this operation")
	ErrPKCERequired                       = errors.New("PKCE is required for a public OAuth client")
	ErrAuthorizationCodeRequired          = errors.New("authorization_code grant is required")
	ErrAuthorizationCodeFlowNotAllowed    = errors.New("OAuth client is not allowed to use authorization code flow")
	ErrRedirectURIAlreadyExists           = errors.New("OAuth client redirect URI already exists")
	ErrRedirectURINotFound                = errors.New("OAuth client redirect URI not found")
	ErrLastRedirectURI                    = errors.New("OAuth client must retain a redirect URI")
	ErrPostLogoutRedirectURIAlreadyExists = errors.New("OAuth client post-logout redirect URI already exists")
	ErrPostLogoutRedirectURINotFound      = errors.New("OAuth client post-logout redirect URI not found")
	ErrGrantTypeAlreadyExists             = errors.New("OAuth client grant type already exists")
	ErrGrantTypeNotFound                  = errors.New("OAuth client grant type not found")
	ErrScopeAlreadyExists                 = errors.New("OAuth client scope already exists")
	ErrScopeNotFound                      = errors.New("OAuth client scope not found")
	ErrCredentialsNotAllowed              = errors.New("credentials are not allowed for a public OAuth client")
	ErrUsableCredentialRequired           = errors.New(
		"a usable credential is required for a confidential OAuth client",
	)
	ErrCredentialNotFound = errors.New("OAuth client credential not found")
	ErrCredentialRevoked  = errors.New("OAuth client credential is revoked")
	ErrCredentialExpired  = errors.New("OAuth client credential is expired")
)
