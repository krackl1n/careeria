package authorizationtransaction

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	openidconnectdomain "github.com/krackl1n/careeria/services/identity/internal/domain/openidconnect"
)

const modelVersion = 1

type model struct {
	Version int `json:"version"`

	ID            string                             `json:"id"`
	OAuthClientID string                             `json:"oauth_client_id"`
	RedirectURI   oauthclientdomain.RedirectURIValue `json:"redirect_uri"`

	RequestedScopes []oauthdomain.Scope        `json:"requested_scopes"`
	State           *domain.State              `json:"state"`
	Nonce           *openidconnectdomain.Nonce `json:"nonce"`
	PKCE            *oauthdomain.PKCE          `json:"pkce"`

	Status   domain.Status   `json:"status"`
	NextStep domain.NextStep `json:"next_step"`

	IdentityID           *string                       `json:"identity_id"`
	AuthenticationLevel  *authenticationdomain.Level   `json:"authentication_level"`
	AuthenticatedMethods []authenticationdomain.Method `json:"authenticated_methods"`
	GrantedScopes        []oauthdomain.Scope           `json:"granted_scopes"`
	AuthorizationSource  *domain.AuthorizationSource   `json:"authorization_source"`
	Verification         *verificationReferenceModel   `json:"verification,omitempty"`

	AuthenticatedAt *time.Time `json:"authenticated_at"`
	ExpiresAt       time.Time  `json:"expires_at"`
}

type verificationReferenceModel struct {
	Channel     domain.VerificationChannel `json:"channel"`
	ContactID   string                     `json:"contact_id"`
	ChallengeID string                     `json:"challenge_id"`
	ExpiresAt   time.Time                  `json:"expires_at"`
}

func toModel(entity *domain.AuthorizationTransaction) model {
	result := model{
		Version: modelVersion,

		ID:            entity.ID.UUID().String(),
		OAuthClientID: entity.OAuthClientID.UUID().String(),
		RedirectURI:   entity.RedirectURI,

		RequestedScopes: entity.RequestedScopes,
		State:           entity.State,
		Nonce:           entity.Nonce,
		PKCE:            entity.PKCE,

		Status:   entity.Status,
		NextStep: entity.NextStep,

		AuthenticationLevel:  entity.AuthenticationLevel,
		AuthenticatedMethods: entity.AuthenticatedMethods,
		GrantedScopes:        entity.GrantedScopes,
		AuthorizationSource:  entity.AuthorizationSource,

		AuthenticatedAt: entity.AuthenticatedAt,
		ExpiresAt:       entity.ExpiresAt,
	}
	if entity.Verification != nil {
		result.Verification = &verificationReferenceModel{
			Channel:     entity.Verification.Channel,
			ContactID:   entity.Verification.ContactID.String(),
			ChallengeID: entity.Verification.ChallengeID.String(),
			ExpiresAt:   entity.Verification.ExpiresAt,
		}
	}

	if entity.IdentityID != nil {
		value := entity.IdentityID.UUID().String()
		result.IdentityID = &value
	}

	return result
}

func (m model) toDomain() (*domain.AuthorizationTransaction, error) {
	if m.Version != modelVersion {
		return nil, fmt.Errorf("unsupported model version %d", m.Version)
	}

	id, err := uuid.Parse(m.ID)
	if err != nil {
		return nil, fmt.Errorf("parse identifier: %w", err)
	}
	oauthClientID, err := uuid.Parse(m.OAuthClientID)
	if err != nil {
		return nil, fmt.Errorf("parse OAuth client identifier: %w", err)
	}

	var identityID *identitydomain.ID
	if m.IdentityID != nil {
		value, err := uuid.Parse(*m.IdentityID)
		if err != nil {
			return nil, fmt.Errorf("parse identity identifier: %w", err)
		}

		parsed := identitydomain.ID(value)
		identityID = &parsed
	}

	var verification *domain.VerificationReference
	if m.Verification != nil {
		channel, err := domain.ParseVerificationChannel(string(m.Verification.Channel))
		if err != nil {
			return nil, fmt.Errorf("parse verification channel: %w", err)
		}
		contactID, err := uuid.Parse(m.Verification.ContactID)
		if err != nil {
			return nil, fmt.Errorf("parse verification contact identifier: %w", err)
		}
		challengeID, err := uuid.Parse(m.Verification.ChallengeID)
		if err != nil {
			return nil, fmt.Errorf("parse verification challenge identifier: %w", err)
		}
		parsed, err := domain.NewVerificationReference(channel, contactID, challengeID, m.Verification.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("parse verification reference: %w", err)
		}
		verification = &parsed
	}

	if m.ExpiresAt.IsZero() {
		return nil, errors.New("expiration is missing")
	}

	return &domain.AuthorizationTransaction{
		ID: domain.ID(id),

		OAuthClientID: oauthclientdomain.ID(oauthClientID),

		RedirectURI:     m.RedirectURI,
		RequestedScopes: m.RequestedScopes,
		State:           m.State,
		Nonce:           m.Nonce,
		PKCE:            m.PKCE,

		Status:   m.Status,
		NextStep: m.NextStep,

		IdentityID:           identityID,
		AuthenticationLevel:  m.AuthenticationLevel,
		AuthenticatedMethods: m.AuthenticatedMethods,
		GrantedScopes:        m.GrantedScopes,
		AuthorizationSource:  m.AuthorizationSource,
		Verification:         verification,

		AuthenticatedAt: m.AuthenticatedAt,
		ExpiresAt:       m.ExpiresAt,
	}, nil
}
