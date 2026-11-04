package oauthclient

import (
	"errors"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type ID uuid.UUID
type CredentialID uuid.UUID
type RedirectURIID uuid.UUID
type PostLogoutRedirectURIID uuid.UUID

func (id ID) UUID() uuid.UUID                      { return uuid.UUID(id) }
func (id CredentialID) UUID() uuid.UUID            { return uuid.UUID(id) }
func (id RedirectURIID) UUID() uuid.UUID           { return uuid.UUID(id) }
func (id PostLogoutRedirectURIID) UUID() uuid.UUID { return uuid.UUID(id) }

func (id ID) IsValid() bool                      { return uuid.UUID(id) != uuid.Nil }
func (id CredentialID) IsValid() bool            { return uuid.UUID(id) != uuid.Nil }
func (id RedirectURIID) IsValid() bool           { return uuid.UUID(id) != uuid.Nil }
func (id PostLogoutRedirectURIID) IsValid() bool { return uuid.UUID(id) != uuid.Nil }

type ClientID string

func NewClientID(value string) (ClientID, error) {
	value, ok := domain.NormalizeToken(value)
	if !ok {
		return "", errors.New("OAuth client identifier must be a non-empty token")
	}

	return ClientID(value), nil
}

type ClientType string

const (
	ClientTypePublic       ClientType = "public"
	ClientTypeConfidential ClientType = "confidential"
)

func ParseClientType(value string) (ClientType, error) {
	clientType := ClientType(value)
	switch clientType {
	case ClientTypePublic, ClientTypeConfidential:
		return clientType, nil
	default:
		return "", errors.New("unknown OAuth client type")
	}
}

type TokenEndpointAuthMethod string

const (
	TokenEndpointAuthMethodNone              TokenEndpointAuthMethod = "none"
	TokenEndpointAuthMethodClientSecretBasic TokenEndpointAuthMethod = "client_secret_basic"
)

func ParseTokenEndpointAuthMethod(value string) (TokenEndpointAuthMethod, error) {
	method := TokenEndpointAuthMethod(value)
	switch method {
	case TokenEndpointAuthMethodNone, TokenEndpointAuthMethodClientSecretBasic:
		return method, nil
	default:
		return "", errors.New("unknown OAuth token endpoint authentication method")
	}
}

type ConsentMode string

const (
	ConsentModeRequired ConsentMode = "required"
	ConsentModeRemember ConsentMode = "remember"
	ConsentModeTrusted  ConsentMode = "trusted"
)

func ParseConsentMode(value string) (ConsentMode, error) {
	mode := ConsentMode(value)
	switch mode {
	case ConsentModeRequired, ConsentModeRemember, ConsentModeTrusted:
		return mode, nil
	default:
		return "", errors.New("unknown OAuth consent mode")
	}
}

type Status string

const (
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
	StatusDeleted  Status = "deleted"
)

func ParseStatus(value string) (Status, error) {
	status := Status(value)
	switch status {
	case StatusActive, StatusDisabled, StatusDeleted:
		return status, nil
	default:
		return "", errors.New("unknown OAuth client status")
	}
}

type SecretHash string

func NewSecretHash(value string) (SecretHash, error) {
	if !domain.HasText(value) {
		return "", errors.New("OAuth client secret hash is required")
	}

	return SecretHash(value), nil
}

type RedirectURIValue string

func NewRedirectURI(value string) (RedirectURIValue, error) {
	normalized, err := parseExactURI(value, "OAuth redirect URI")
	if err != nil {
		return "", err
	}

	return RedirectURIValue(normalized), nil
}

type PostLogoutRedirectURIValue string

func NewPostLogoutRedirectURI(value string) (PostLogoutRedirectURIValue, error) {
	normalized, err := parseExactURI(value, "OAuth post-logout redirect URI")
	if err != nil {
		return "", err
	}

	return PostLogoutRedirectURIValue(normalized), nil
}

type GrantType string

const (
	GrantTypeAuthorizationCode GrantType = "authorization_code"
	GrantTypeRefreshToken      GrantType = "refresh_token"
)

func ParseGrantType(value string) (GrantType, error) {
	grantType := GrantType(value)
	switch grantType {
	case GrantTypeAuthorizationCode, GrantTypeRefreshToken:
		return grantType, nil
	default:
		return "", errors.New("unknown OAuth grant type")
	}
}
