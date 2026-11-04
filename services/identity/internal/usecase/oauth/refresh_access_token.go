package oauth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	refreshtokendomain "github.com/krackl1n/careeria/services/identity/internal/domain/refreshtokenfamily"
	securityeventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/securityevent"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
	accesstokenservice "github.com/krackl1n/careeria/services/identity/internal/service/accesstoken"
)

type RefreshAccessTokenInput struct {
	ClientID     string
	ClientSecret *string
	RefreshToken string
}

type RefreshAccessTokenOutput struct {
	AccessToken  string
	TokenType    string
	ExpiresIn    uint32
	RefreshToken string
	Scope        string
}

type refreshAccessTokenCommand struct {
	clientID     oauthclientdomain.ClientID
	clientSecret *string
	refreshToken string
}

func (s *Service) RefreshAccessToken(
	ctx context.Context,
	input RefreshAccessTokenInput,
) (RefreshAccessTokenOutput, error) {
	command, err := prepareRefreshAccessToken(input)
	if err != nil {
		return RefreshAccessTokenOutput{}, invalidTokenRequest(err)
	}

	var output RefreshAccessTokenOutput
	var committedError error
	var auditClientID *oauthclientdomain.ID
	var auditIdentityID *identitydomain.ID
	var auditSessionID *sessiondomain.ID
	err = s.tx.Do(ctx, func(txCtx context.Context) error {
		now := s.clock.Now()
		client, err := s.getAndAuthenticateClient(txCtx, command.clientID, command.clientSecret, now)
		if err != nil {
			return err
		}
		auditClientID = &client.ID
		if err := client.ValidateTokenExchange(oauthclientdomain.GrantTypeRefreshToken, now); err != nil {
			return clientGrantError(err)
		}

		presentedHash := refreshtokendomain.TokenHash(s.tokenGenerator.Hash(command.refreshToken))
		family, err := s.refreshTokenFamilies.GetByTokenHashForUpdate(txCtx, presentedHash)
		if err != nil {
			if errors.Is(err, refreshtokendomain.ErrTokenNotFound) || errors.Is(err, refreshtokendomain.ErrNotFound) {
				return invalidGrant(err)
			}
			return fmt.Errorf("get refresh token family: %w", err)
		}
		presentedTokenID, err := family.TokenIDByHash(presentedHash)
		if err != nil {
			return invalidGrant(err)
		}

		session, err := s.sessions.Get(txCtx, family.SessionID)
		if err != nil {
			if errors.Is(err, sessiondomain.ErrNotFound) {
				return invalidGrant(err)
			}
			return fmt.Errorf("get OAuth session: %w", err)
		}
		if session.OAuthClientID != client.ID {
			return invalidGrant(errors.New("refresh token belongs to another OAuth client"))
		}
		if err := session.ValidateForUse(now); err != nil {
			return invalidGrant(err)
		}
		auditIdentityID = &session.IdentityID
		auditSessionID = &session.ID

		raw, nextHash, err := s.tokenGenerator.New()
		if err != nil {
			return fmt.Errorf("generate refresh token: %w", err)
		}
		nextTokenID := refreshtokendomain.TokenID(s.ids.New())
		outcome, err := family.Rotate(
			presentedTokenID,
			nextTokenID,
			refreshtokendomain.TokenHash(nextHash),
			domain.MinimumTime(now.Add(s.cfg.RefreshTokenLifetime), family.AbsoluteExpiresAt),
			nil,
			nil,
			now,
		)
		if err != nil {
			return invalidGrant(err)
		}
		if outcome == refreshtokendomain.RotationOutcomeReuseDetected {
			if err := session.Revoke("refresh_token_reuse_detected", now); err != nil {
				return fmt.Errorf("revoke reused refresh token session: %w", err)
			}
			if err := s.refreshTokenFamilies.Update(txCtx, family); err != nil {
				return fmt.Errorf("save reused refresh token family: %w", err)
			}
			if err := s.sessions.Update(txCtx, session); err != nil {
				return fmt.Errorf("save revoked OAuth session: %w", err)
			}
			reuseDetected, err := s.events.Create(refreshtokendomain.EventTypeReuseDetected, refreshtokendomain.AggregateType, family.ID.UUID(), nil, refreshtokendomain.ReuseDetectedEvent{
				FamilyID:      family.ID.UUID(),
				SessionID:     family.SessionID.UUID(),
				ReusedTokenID: presentedTokenID.UUID(),
				DetectedAt:    now,
				RevokedAt:     *family.RevokedAt,
			}, now)
			if err != nil {
				return fmt.Errorf("create refresh token reuse detected event: %w", err)
			}
			if err := s.outbox.Create(txCtx, reuseDetected); err != nil {
				return fmt.Errorf("save refresh token reuse detected event to outbox: %w", err)
			}
			sessionRevoked, err := s.events.Create(sessiondomain.EventTypeRevoked, sessiondomain.AggregateType, session.ID.UUID(), nil, sessiondomain.RevokedEvent{
				SessionID:     session.ID.UUID(),
				IdentityID:    session.IdentityID.UUID(),
				OAuthClientID: session.OAuthClientID.UUID(),
				Reason:        *session.RevokeReason,
				RevokedAt:     *session.RevokedAt,
			}, now)
			if err != nil {
				return fmt.Errorf("create session revoked event: %w", err)
			}
			if err := s.outbox.Create(txCtx, sessionRevoked); err != nil {
				return fmt.Errorf("save session revoked event to outbox: %w", err)
			}
			actor, err := securityeventdomain.NewIdentityActor(session.IdentityID)
			if err != nil {
				return fmt.Errorf("create refresh token reuse actor: %w", err)
			}
			if err := s.saveSecurityEvent(txCtx, securityEventInput{
				Type:          securityeventdomain.EventTypeRefreshTokenExchange,
				Result:        securityeventdomain.ResultBlocked,
				Actor:         actor,
				SessionID:     &session.ID,
				OAuthClientID: &client.ID,
				Metadata: map[string]any{
					"grant_type": "refresh_token",
					"reason":     "refresh_token_reuse_detected",
				},
				OccurredAt: now,
			}); err != nil {
				return err
			}
			committedError = invalidGrant(errors.New("refresh token reuse detected"))
			return nil
		}

		if err := session.RecordActivity(
			now,
			domain.MinimumTime(now.Add(s.cfg.SessionIdleLifetime), session.AbsoluteExpiresAt),
		); err != nil {
			return invalidGrant(err)
		}
		accessToken, err := s.accessTokens.Issue(txCtx, accesstokenservice.IssueInput{
			IdentityID:           session.IdentityID,
			SessionID:            session.ID,
			ClientID:             client.ClientID,
			GrantedScopes:        session.Scopes,
			AuthenticationLevel:  session.AuthenticationLevel,
			AuthenticatedMethods: session.AuthenticatedMethods,
			AuthenticatedAt:      session.AuthenticatedAt,
		})
		if err != nil {
			return fmt.Errorf("issue OAuth access token: %w", err)
		}
		if err := s.refreshTokenFamilies.Update(txCtx, family); err != nil {
			return fmt.Errorf("save rotated refresh token family: %w", err)
		}
		if err := s.sessions.Update(txCtx, session); err != nil {
			return fmt.Errorf("save OAuth session activity: %w", err)
		}

		scopes := make([]string, len(session.Scopes))
		for index := range session.Scopes {
			scopes[index] = string(session.Scopes[index])
		}
		newToken := family.Tokens[len(family.Tokens)-1]
		tokenRotated, err := s.events.Create(refreshtokendomain.EventTypeTokenRotated, refreshtokendomain.AggregateType, family.ID.UUID(), nil, refreshtokendomain.TokenRotatedEvent{
			FamilyID:          family.ID.UUID(),
			SessionID:         family.SessionID.UUID(),
			PreviousTokenID:   presentedTokenID.UUID(),
			NewTokenID:        nextTokenID.UUID(),
			NewTokenExpiresAt: newToken.ExpiresAt,
			CreatedFromIP:     newToken.CreatedFromIP,
			CreatedUserAgent:  newToken.CreatedUserAgent,
			RotatedAt:         now,
		}, now)
		if err != nil {
			return fmt.Errorf("create refresh token rotated event: %w", err)
		}
		if err := s.outbox.Create(txCtx, tokenRotated); err != nil {
			return fmt.Errorf("save refresh token rotated event to outbox: %w", err)
		}
		actor, err := securityeventdomain.NewIdentityActor(session.IdentityID)
		if err != nil {
			return fmt.Errorf("create refresh token exchange actor: %w", err)
		}
		if err := s.saveSecurityEvent(txCtx, securityEventInput{
			Type:          securityeventdomain.EventTypeRefreshTokenExchange,
			Result:        securityeventdomain.ResultSuccess,
			Actor:         actor,
			SessionID:     &session.ID,
			OAuthClientID: &client.ID,
			Metadata: map[string]any{
				"grant_type": "refresh_token",
				"scopes":     scopes,
			},
			OccurredAt: now,
		}); err != nil {
			return err
		}
		output = RefreshAccessTokenOutput{
			AccessToken:  accessToken.Value,
			TokenType:    "Bearer",
			ExpiresIn:    uint32(accessToken.ExpiresAt.Sub(accessToken.IssuedAt).Seconds()),
			RefreshToken: raw,
			Scope:        strings.Join(scopes, " "),
		}
		return nil
	})
	if err != nil {
		if !isOAuthFailure(err) {
			return RefreshAccessTokenOutput{}, err
		}
		actor := securityeventdomain.NewAnonymousActor()
		if auditIdentityID != nil {
			actor, _ = securityeventdomain.NewIdentityActor(*auditIdentityID)
		}
		if auditErr := s.saveSecurityEvent(ctx, securityEventInput{
			Type:          securityeventdomain.EventTypeRefreshTokenExchange,
			Result:        securityeventdomain.ResultFailure,
			Actor:         actor,
			SessionID:     auditSessionID,
			OAuthClientID: auditClientID,
			Metadata: map[string]any{
				"grant_type": "refresh_token",
				"reason":     securityFailureReason(err),
			},
			OccurredAt: s.clock.Now(),
		}); auditErr != nil {
			return RefreshAccessTokenOutput{}, errors.Join(err, auditErr)
		}
		return RefreshAccessTokenOutput{}, err
	}
	if committedError != nil {
		return RefreshAccessTokenOutput{}, committedError
	}

	return output, nil
}

func prepareRefreshAccessToken(input RefreshAccessTokenInput) (refreshAccessTokenCommand, error) {
	clientID, err := oauthclientdomain.NewClientID(input.ClientID)
	if err != nil {
		return refreshAccessTokenCommand{}, err
	}
	if input.RefreshToken == "" {
		return refreshAccessTokenCommand{}, errors.New("refresh token is required")
	}

	return refreshAccessTokenCommand{
		clientID:     clientID,
		clientSecret: input.ClientSecret,
		refreshToken: input.RefreshToken,
	}, nil
}
