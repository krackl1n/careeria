package idtoken

import (
	"context"
	"errors"
	"fmt"
	"time"

	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	openidconnectdomain "github.com/krackl1n/careeria/services/identity/internal/domain/openidconnect"
)

type IssueInput struct {
	IdentityID identitydomain.ID
	ClientID   oauthclientdomain.ClientID
	Nonce      *openidconnectdomain.Nonce
	Identity   IdentityClaims

	AuthenticationLevel  authenticationdomain.Level
	AuthenticatedMethods []authenticationdomain.Method
	AuthenticatedAt      time.Time
}

type IssuedToken struct {
	Value     string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

func (s *Service) Issue(ctx context.Context, input IssueInput) (IssuedToken, error) {
	now := s.clock.Now()
	issuedAt := time.Unix(now.Unix(), 0).UTC()
	expiresAt := issuedAt.Add(s.cfg.Lifetime)
	key, err := s.keys.GetActive(ctx, s.algorithm, now)
	if err != nil {
		return IssuedToken{}, fmt.Errorf("issue ID token: get active signing key: %w", err)
	}
	if key.ExpiresAt != nil && key.ExpiresAt.Before(expiresAt) {
		return IssuedToken{}, errors.New("issue ID token: signing key expires before ID token")
	}

	var nonce *string
	if input.Nonce != nil {
		value := string(*input.Nonce)
		nonce = &value
	}
	methods := make([]string, len(input.AuthenticatedMethods))
	for index := range input.AuthenticatedMethods {
		methods[index] = string(input.AuthenticatedMethods[index])
	}
	value, err := s.signer.SignIDToken(ctx, SignInput{
		KeyID:         key.KeyID,
		Algorithm:     key.Algorithm,
		PrivateKeyRef: key.PrivateKeyRef,
		Claims: Claims{
			Issuer:               s.cfg.Issuer,
			Subject:              input.IdentityID.UUID().String(),
			Audience:             string(input.ClientID),
			ExpiresAt:            expiresAt,
			IssuedAt:             issuedAt,
			Nonce:                nonce,
			Identity:             input.Identity,
			AuthenticationLevel:  string(input.AuthenticationLevel),
			AuthenticatedMethods: methods,
			AuthenticatedAt:      input.AuthenticatedAt,
		},
	})
	if err != nil {
		return IssuedToken{}, fmt.Errorf("issue ID token: sign: %w", err)
	}

	return IssuedToken{
		Value:     value,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}, nil
}
