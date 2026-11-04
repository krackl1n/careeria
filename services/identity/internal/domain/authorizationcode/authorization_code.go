package authorizationcode

import (
	"errors"
	"fmt"
	"slices"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	openidconnectdomain "github.com/krackl1n/careeria/services/identity/internal/domain/openidconnect"
)

type AuthorizationCode struct {
	Hash Hash

	OAuthClientID oauthclientdomain.ID
	IdentityID    identitydomain.ID

	RedirectURI oauthclientdomain.RedirectURIValue
	Scopes      []oauthdomain.Scope
	Nonce       *openidconnectdomain.Nonce
	PKCE        *oauthdomain.PKCE

	AuthenticationLevel  authenticationdomain.Level
	AuthenticatedMethods []authenticationdomain.Method

	AuthenticatedAt time.Time
	IssuedAt        time.Time
	ExpiresAt       time.Time
}

func New(
	hash Hash,
	oauthClientID oauthclientdomain.ID,
	redirectURI oauthclientdomain.RedirectURIValue,
	identityID identitydomain.ID,
	authenticationLevel authenticationdomain.Level,
	authenticatedMethods []authenticationdomain.Method,
	scopes []oauthdomain.Scope,
	nonce *openidconnectdomain.Nonce,
	pkce *oauthdomain.PKCE,
	authenticatedAt time.Time,
	expiresAt time.Time,
	now time.Time,
) (*AuthorizationCode, error) {
	entity := &AuthorizationCode{
		Hash: hash,

		OAuthClientID: oauthClientID,
		IdentityID:    identityID,

		RedirectURI: redirectURI,
		Scopes:      slices.Clone(scopes),
		Nonce:       domain.ClonePointer(nonce),
		PKCE:        domain.ClonePointer(pkce),

		AuthenticationLevel:  authenticationLevel,
		AuthenticatedMethods: slices.Clone(authenticatedMethods),

		AuthenticatedAt: authenticatedAt,
		IssuedAt:        now,
		ExpiresAt:       expiresAt,
	}
	if err := entity.ensureIssuable(); err != nil {
		return nil, fmt.Errorf("create authorization code: %w", err)
	}

	return entity, nil
}

func (c *AuthorizationCode) ValidateForExchange(
	oauthClientID oauthclientdomain.ID,
	redirectURI oauthclientdomain.RedirectURIValue,
	verifier *oauthdomain.CodeVerifier,
	now time.Time,
) error {
	if c == nil {
		return errors.New("validate authorization code for exchange: authorization code is nil")
	}
	if now.IsZero() || now.Before(c.IssuedAt) {
		return errors.New("validate authorization code for exchange: time is invalid")
	}
	if !now.Before(c.ExpiresAt) {
		return fmt.Errorf("validate authorization code for exchange: %w", ErrExpired)
	}
	if oauthClientID != c.OAuthClientID {
		return fmt.Errorf("validate authorization code for exchange: %w", ErrOAuthClientMismatch)
	}
	if redirectURI != c.RedirectURI {
		return fmt.Errorf("validate authorization code for exchange: %w", ErrRedirectURIMismatch)
	}
	if c.PKCE == nil {
		return nil
	}
	if verifier == nil {
		return fmt.Errorf("validate authorization code for exchange: %w", ErrPKCERequired)
	}
	if !c.PKCE.Verify(*verifier) {
		return fmt.Errorf("validate authorization code for exchange: %w", ErrPKCEVerificationFailed)
	}

	return nil
}

func (c *AuthorizationCode) ensureIssuable() error {
	if c == nil {
		return errors.New("authorization code is nil")
	}
	if !c.OAuthClientID.IsValid() {
		return errors.New("authorization code OAuth client identifier is invalid")
	}
	if !c.IdentityID.IsValid() {
		return errors.New("authorization code identity identifier is invalid")
	}
	resolvedLevel, err := authenticationdomain.ResolveLevel(c.AuthenticatedMethods)
	if err != nil {
		return fmt.Errorf("authorization code authentication snapshot is invalid: %w", err)
	}
	if !c.AuthenticationLevel.IsValid() || c.AuthenticationLevel != resolvedLevel {
		return errors.New("authorization code authentication level does not match methods")
	}
	if domain.HasDuplicates(c.Scopes) {
		return errors.New("authorization code scope is duplicated")
	}
	if c.AuthenticatedAt.IsZero() || c.IssuedAt.Before(c.AuthenticatedAt) {
		return errors.New("authorization code authentication time is invalid")
	}
	if c.IssuedAt.IsZero() || !c.ExpiresAt.After(c.IssuedAt) {
		return errors.New("authorization code expiration state is invalid")
	}

	return nil
}
