package accesstoken

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

type IssueInput struct {
	IdentityID identitydomain.ID
	SessionID  sessiondomain.ID
	ClientID   oauthclientdomain.ClientID

	GrantedScopes []oauthdomain.Scope

	AuthenticationLevel  authenticationdomain.Level
	AuthenticatedMethods []authenticationdomain.Method
	AuthenticatedAt      time.Time
}

type IssuedToken struct {
	Value     string
	ID        uuid.UUID
	IssuedAt  time.Time
	ExpiresAt time.Time
}

func (s *Service) Issue(ctx context.Context, input IssueInput) (IssuedToken, error) {
	now := s.clock.Now()
	issuedAt := time.Unix(now.Unix(), 0).UTC()
	expiresAt := issuedAt.Add(s.cfg.Lifetime)
	key, err := s.keys.GetActive(ctx, s.algorithm, now)
	if err != nil {
		return IssuedToken{}, fmt.Errorf("issue access token: get active signing key: %w", err)
	}
	if key.ExpiresAt != nil && key.ExpiresAt.Before(expiresAt) {
		return IssuedToken{}, errors.New("issue access token: signing key expires before access token")
	}

	tokenID := s.ids.New()

	claims := Claims{
		Issuer:   s.cfg.Issuer,
		Subject:  input.IdentityID.UUID().String(),
		Audience: append([]string(nil), s.cfg.Audience...),

		ExpiresAt: expiresAt,
		IssuedAt:  issuedAt,
		TokenID:   tokenID.String(),

		ClientID:  string(input.ClientID),
		Scopes:    scopesToStrings(input.GrantedScopes),
		SessionID: input.SessionID.UUID().String(),

		AuthenticationLevel:  string(input.AuthenticationLevel),
		AuthenticatedMethods: authenticationMethodsToStrings(input.AuthenticatedMethods),
		AuthenticatedAt:      input.AuthenticatedAt,
	}

	value, err := s.signer.Sign(ctx, SignInput{
		KeyID:         key.KeyID,
		Algorithm:     key.Algorithm,
		PrivateKeyRef: key.PrivateKeyRef,
		Claims:        claims,
	})
	if err != nil {
		return IssuedToken{}, fmt.Errorf("issue access token: sign: %w", err)
	}

	return IssuedToken{
		Value:     value,
		ID:        tokenID,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}, nil
}

func scopesToStrings(scopes []oauthdomain.Scope) []string {
	values := make([]string, len(scopes))
	for i, scope := range scopes {
		values[i] = string(scope)
	}
	return values
}

func authenticationMethodsToStrings(methods []authenticationdomain.Method) []string {
	values := make([]string, len(methods))
	for i, method := range methods {
		values[i] = string(method)
	}
	return values
}
