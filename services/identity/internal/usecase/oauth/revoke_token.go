package oauth

import (
	"context"
	"errors"
	"fmt"
	"time"

	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	refreshtokendomain "github.com/krackl1n/careeria/services/identity/internal/domain/refreshtokenfamily"
	securityeventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/securityevent"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
	accesstokenservice "github.com/krackl1n/careeria/services/identity/internal/service/accesstoken"
)

const tokenRevokedReason = "oauth_token_revoked"

type RevokeTokenInput struct {
	ClientID      string
	ClientSecret  *string
	Token         string
	TokenTypeHint *string
}

type revokeTokenCommand struct {
	clientID      oauthclientdomain.ClientID
	clientSecret  *string
	token         string
	tokenTypeHint *string
}

type tokenRevocation struct {
	recognized bool
	owned      bool
	changed    bool
	identityID *identitydomain.ID
	sessionID  *sessiondomain.ID
}

func (s *Service) RevokeToken(ctx context.Context, input RevokeTokenInput) error {
	command, err := prepareRevokeToken(input)
	if err != nil {
		return invalidRevocationRequest(err)
	}

	var auditClientID *oauthclientdomain.ID
	err = s.tx.Do(ctx, func(txCtx context.Context) error {
		now := s.clock.Now()
		client, err := s.getAndAuthenticateClient(txCtx, command.clientID, command.clientSecret, now)
		if err != nil {
			return err
		}
		auditClientID = &client.ID

		tokenTypeHint := ""
		if command.tokenTypeHint != nil {
			tokenTypeHint = *command.tokenTypeHint
		}
		var revocation tokenRevocation
		switch tokenTypeHint {
		case "access_token":
			revocation, err = s.revokeAccessToken(txCtx, client, command.token, now)
			if err == nil && !revocation.recognized {
				revocation, err = s.revokeRefreshToken(txCtx, client, command.token, now)
			}
		case "refresh_token":
			revocation, err = s.revokeRefreshToken(txCtx, client, command.token, now)
			if err == nil && !revocation.recognized {
				revocation, err = s.revokeAccessToken(txCtx, client, command.token, now)
			}
		default:
			revocation, err = s.revokeRefreshToken(txCtx, client, command.token, now)
			if err == nil && !revocation.recognized {
				revocation, err = s.revokeAccessToken(txCtx, client, command.token, now)
			}
		}
		if err != nil {
			return err
		}

		actor := securityeventdomain.NewAnonymousActor()
		if revocation.identityID != nil {
			actor, _ = securityeventdomain.NewIdentityActor(*revocation.identityID)
		}
		result := securityeventdomain.ResultFailure
		if revocation.owned {
			result = securityeventdomain.ResultSuccess
		}
		return s.saveSecurityEvent(txCtx, securityEventInput{
			Type:          securityeventdomain.EventTypeTokenRevocation,
			Result:        result,
			Actor:         actor,
			SessionID:     revocation.sessionID,
			OAuthClientID: &client.ID,
			Metadata: map[string]any{
				"token_type_hint": tokenTypeHint,
				"recognized":      revocation.recognized,
				"owned":           revocation.owned,
				"changed":         revocation.changed,
			},
			OccurredAt: now,
		})
	})
	if err != nil {
		if !isOAuthFailure(err) {
			return err
		}
		if auditErr := s.saveSecurityEvent(ctx, securityEventInput{
			Type:          securityeventdomain.EventTypeTokenRevocation,
			Result:        securityeventdomain.ResultFailure,
			Actor:         securityeventdomain.NewAnonymousActor(),
			OAuthClientID: auditClientID,
			Metadata: map[string]any{
				"reason": securityFailureReason(err),
			},
			OccurredAt: s.clock.Now(),
		}); auditErr != nil {
			return errors.Join(err, auditErr)
		}
		return err
	}

	return nil
}

func prepareRevokeToken(input RevokeTokenInput) (revokeTokenCommand, error) {
	clientID, err := oauthclientdomain.NewClientID(input.ClientID)
	if err != nil {
		return revokeTokenCommand{}, err
	}
	if input.Token == "" {
		return revokeTokenCommand{}, errors.New("token is required")
	}

	return revokeTokenCommand{
		clientID:      clientID,
		clientSecret:  input.ClientSecret,
		token:         input.Token,
		tokenTypeHint: input.TokenTypeHint,
	}, nil
}

func (s *Service) revokeRefreshToken(
	ctx context.Context,
	client *oauthclientdomain.OAuthClient,
	rawToken string,
	now time.Time,
) (tokenRevocation, error) {
	hash := refreshtokendomain.TokenHash(s.tokenGenerator.Hash(rawToken))
	family, err := s.refreshTokenFamilies.GetByTokenHashForUpdate(ctx, hash)
	if err != nil {
		if errors.Is(err, refreshtokendomain.ErrTokenNotFound) || errors.Is(err, refreshtokendomain.ErrNotFound) {
			return tokenRevocation{}, nil
		}
		return tokenRevocation{}, fmt.Errorf("find refresh token for revocation: %w", err)
	}

	session, err := s.sessions.Get(ctx, family.SessionID)
	if err != nil {
		return tokenRevocation{recognized: true}, fmt.Errorf("get refresh token session for revocation: %w", err)
	}
	if session.OAuthClientID != client.ID {
		return tokenRevocation{recognized: true}, nil
	}
	result := tokenRevocation{
		recognized: true,
		owned:      true,
		identityID: &session.IdentityID,
		sessionID:  &session.ID,
	}
	if family.RevokedAt != nil {
		return result, nil
	}
	if err := family.Revoke(tokenRevokedReason, now); err != nil {
		return result, fmt.Errorf("revoke refresh token family: %w", err)
	}
	if err := s.refreshTokenFamilies.Update(ctx, family); err != nil {
		return result, fmt.Errorf("save revoked refresh token family: %w", err)
	}
	familyRevoked, err := s.events.Create(refreshtokendomain.EventTypeRevoked, refreshtokendomain.AggregateType, family.ID.UUID(), nil, refreshtokendomain.RevokedEvent{
		FamilyID:  family.ID.UUID(),
		SessionID: family.SessionID.UUID(),
		Reason:    *family.RevokeReason,
		RevokedAt: *family.RevokedAt,
	}, now)
	if err != nil {
		return result, fmt.Errorf("create refresh token family revoked event: %w", err)
	}
	if err := s.outbox.Create(ctx, familyRevoked); err != nil {
		return result, fmt.Errorf("save refresh token family revoked event to outbox: %w", err)
	}
	result.changed = true

	return result, nil
}

func (s *Service) revokeAccessToken(
	ctx context.Context,
	client *oauthclientdomain.OAuthClient,
	rawToken string,
	now time.Time,
) (tokenRevocation, error) {
	token, err := s.accessTokenVerifier.Verify(ctx, rawToken)
	if err != nil {
		if errors.Is(err, accesstokenservice.ErrInvalidToken) {
			return tokenRevocation{}, nil
		}
		return tokenRevocation{}, fmt.Errorf("verify access token for revocation: %w", err)
	}
	if token.ClientID != client.ClientID {
		return tokenRevocation{recognized: true}, nil
	}

	session, err := s.sessions.Get(ctx, token.SessionID)
	if err != nil {
		if errors.Is(err, sessiondomain.ErrNotFound) {
			return tokenRevocation{recognized: true}, nil
		}
		return tokenRevocation{recognized: true}, fmt.Errorf("get access token session for revocation: %w", err)
	}
	if session.IdentityID != token.IdentityID || session.OAuthClientID != client.ID {
		return tokenRevocation{recognized: true}, nil
	}
	result := tokenRevocation{
		recognized: true,
		owned:      true,
		identityID: &session.IdentityID,
		sessionID:  &session.ID,
	}
	if session.RevokedAt != nil {
		return result, nil
	}
	if err := session.Revoke(tokenRevokedReason, now); err != nil {
		return result, fmt.Errorf("revoke access token session: %w", err)
	}
	if err := s.sessions.Update(ctx, session); err != nil {
		return result, fmt.Errorf("save revoked access token session: %w", err)
	}
	sessionRevoked, err := s.events.Create(sessiondomain.EventTypeRevoked, sessiondomain.AggregateType, session.ID.UUID(), nil, sessiondomain.RevokedEvent{
		SessionID:     session.ID.UUID(),
		IdentityID:    session.IdentityID.UUID(),
		OAuthClientID: session.OAuthClientID.UUID(),
		Reason:        *session.RevokeReason,
		RevokedAt:     *session.RevokedAt,
	}, now)
	if err != nil {
		return result, fmt.Errorf("create session revoked event: %w", err)
	}
	if err := s.outbox.Create(ctx, sessionRevoked); err != nil {
		return result, fmt.Errorf("save session revoked event to outbox: %w", err)
	}
	result.changed = true

	return result, nil
}
