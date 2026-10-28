package authorizationcode

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationcode"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	openidconnectdomain "github.com/krackl1n/careeria/services/identity/internal/domain/openidconnect"
)

const modelVersion = 1

type model struct {
	Version int `json:"version"`

	Hash          domain.Hash `json:"hash"`
	OAuthClientID string      `json:"oauth_client_id"`
	IdentityID    string      `json:"identity_id"`

	RedirectURI oauthclientdomain.RedirectURIValue `json:"redirect_uri"`
	Scopes      []oauthdomain.Scope                `json:"scopes"`
	Nonce       *openidconnectdomain.Nonce         `json:"nonce"`
	PKCE        *oauthdomain.PKCE                  `json:"pkce"`

	AuthenticationLevel  authenticationdomain.Level    `json:"authentication_level"`
	AuthenticatedMethods []authenticationdomain.Method `json:"authenticated_methods"`

	AuthenticatedAt time.Time `json:"authenticated_at"`
	IssuedAt        time.Time `json:"issued_at"`
	ExpiresAt       time.Time `json:"expires_at"`
}

func toModel(entity *domain.AuthorizationCode) model {
	return model{
		Version: modelVersion,

		Hash:          entity.Hash,
		OAuthClientID: entity.OAuthClientID.UUID().String(),
		IdentityID:    entity.IdentityID.UUID().String(),

		RedirectURI: entity.RedirectURI,
		Scopes:      entity.Scopes,
		Nonce:       entity.Nonce,
		PKCE:        entity.PKCE,

		AuthenticationLevel:  entity.AuthenticationLevel,
		AuthenticatedMethods: entity.AuthenticatedMethods,

		AuthenticatedAt: entity.AuthenticatedAt,
		IssuedAt:        entity.IssuedAt,
		ExpiresAt:       entity.ExpiresAt,
	}
}

func (m model) toDomain() (*domain.AuthorizationCode, error) {
	if m.Version != modelVersion {
		return nil, fmt.Errorf("unsupported model version %d", m.Version)
	}

	oauthClientID, err := uuid.Parse(m.OAuthClientID)
	if err != nil {
		return nil, fmt.Errorf("parse OAuth client identifier: %w", err)
	}
	identityID, err := uuid.Parse(m.IdentityID)
	if err != nil {
		return nil, fmt.Errorf("parse identity identifier: %w", err)
	}
	if m.ExpiresAt.IsZero() {
		return nil, errors.New("expiration is missing")
	}

	return &domain.AuthorizationCode{
		Hash: m.Hash,

		OAuthClientID: oauthclientdomain.ID(oauthClientID),
		IdentityID:    identitydomain.ID(identityID),

		RedirectURI: m.RedirectURI,
		Scopes:      m.Scopes,
		Nonce:       m.Nonce,
		PKCE:        m.PKCE,

		AuthenticationLevel:  m.AuthenticationLevel,
		AuthenticatedMethods: m.AuthenticatedMethods,

		AuthenticatedAt: m.AuthenticatedAt,
		IssuedAt:        m.IssuedAt,
		ExpiresAt:       m.ExpiresAt,
	}, nil
}
