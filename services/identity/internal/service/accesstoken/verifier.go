package accesstoken

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
)

func (s *Service) Verify(ctx context.Context, value string) (VerifiedToken, error) {
	keyID, err := s.verifier.KeyID(value)
	if err != nil {
		return VerifiedToken{}, invalidToken(err)
	}

	key, err := s.keys.GetByKeyID(ctx, keyID)
	if err != nil {
		if errors.Is(err, signingkeydomain.ErrNotFound) {
			return VerifiedToken{}, invalidToken(err)
		}
		return VerifiedToken{}, fmt.Errorf("verify access token: get signing key: %w", err)
	}

	now := s.clock.Now()
	if key.Algorithm != s.algorithm {
		return VerifiedToken{}, invalidToken(errors.New("signing algorithm does not match access token configuration"))
	}
	if err := key.ValidateForVerification(now); err != nil {
		return VerifiedToken{}, invalidToken(err)
	}

	claims, err := s.verifier.Verify(VerifyInput{
		Value:     value,
		KeyID:     key.KeyID,
		Algorithm: key.Algorithm,
		PublicJWK: key.PublicJWK,
		Issuer:    s.cfg.Issuer,
		Audience:  append([]string(nil), s.cfg.Audience...),
		Now:       now,
	})
	if err != nil {
		return VerifiedToken{}, invalidToken(err)
	}

	return verifiedTokenFromClaims(claims)
}

func verifiedTokenFromClaims(claims Claims) (VerifiedToken, error) {
	tokenID, err := uuid.Parse(claims.TokenID)
	if err != nil || tokenID == uuid.Nil {
		return VerifiedToken{}, invalidToken(errors.New("token ID is invalid"))
	}
	identityID, err := uuid.Parse(claims.Subject)
	if err != nil || identityID == uuid.Nil {
		return VerifiedToken{}, invalidToken(errors.New("subject is invalid"))
	}
	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil || sessionID == uuid.Nil {
		return VerifiedToken{}, invalidToken(errors.New("session ID is invalid"))
	}
	clientID, err := oauthclientdomain.NewClientID(claims.ClientID)
	if err != nil {
		return VerifiedToken{}, invalidToken(err)
	}

	scopes := make([]oauthdomain.Scope, 0, len(claims.Scopes))
	for _, value := range claims.Scopes {
		scope, err := oauthdomain.NewScope(value)
		if err != nil {
			return VerifiedToken{}, invalidToken(err)
		}
		scopes = append(scopes, scope)
	}

	return VerifiedToken{
		ID:         tokenID,
		IdentityID: identitydomain.ID(identityID),
		SessionID:  sessiondomain.ID(sessionID),
		ClientID:   clientID,
		Scopes:     scopes,
	}, nil
}

func invalidToken(cause error) error {
	return fmt.Errorf("%w: %s", ErrInvalidToken, strings.TrimSpace(cause.Error()))
}
