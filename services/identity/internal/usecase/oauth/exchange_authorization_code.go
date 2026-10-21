package oauth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	authorizationcodedomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationcode"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	refreshtokendomain "github.com/krackl1n/careeria/services/identity/internal/domain/refreshtokenfamily"
	securityeventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/securityevent"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
	accesstokenservice "github.com/krackl1n/careeria/services/identity/internal/service/accesstoken"
	idtokenservice "github.com/krackl1n/careeria/services/identity/internal/service/idtoken"
)

type ExchangeAuthorizationCodeInput struct {
	ClientID     string
	ClientSecret *string
	Code         string
	RedirectURI  string
	CodeVerifier *string
}

type ExchangeAuthorizationCodeOutput struct {
	AccessToken  string
	TokenType    string
	ExpiresIn    uint32
	RefreshToken *string
	IDToken      string
	Scope        string
}

type exchangeAuthorizationCodeCommand struct {
	clientID     oauthclientdomain.ClientID
	clientSecret *string
	code         string
	redirectURI  oauthclientdomain.RedirectURIValue
	codeVerifier *oauthdomain.CodeVerifier
}

func (s *Service) ExchangeAuthorizationCode(
	ctx context.Context,
	input ExchangeAuthorizationCodeInput,
) (ExchangeAuthorizationCodeOutput, error) {
	command, err := prepareExchangeAuthorizationCode(input)
	if err != nil {
		return ExchangeAuthorizationCodeOutput{}, invalidTokenRequest(err)
	}

	var output ExchangeAuthorizationCodeOutput
	var auditClientID *oauthclientdomain.ID
	var auditIdentityID *identitydomain.ID
	err = s.tx.Do(ctx, func(txCtx context.Context) error {
		now := s.clock.Now()
		client, err := s.getAndAuthenticateClient(txCtx, command.clientID, command.clientSecret, now)
		if err != nil {
			return err
		}
		auditClientID = &client.ID
		if err := client.ValidateTokenExchange(oauthclientdomain.GrantTypeAuthorizationCode, now); err != nil {
			return clientGrantError(err)
		}

		code, err := s.authorizationCode.Consume(
			txCtx,
			authorizationcodedomain.Hash(s.tokenGenerator.Hash(command.code)),
		)
		if err != nil {
			if errors.Is(err, authorizationcodedomain.ErrNotFound) {
				return invalidGrant(err)
			}
			return fmt.Errorf("consume authorization code: %w", err)
		}
		if err := code.ValidateForExchange(client.ID, command.redirectURI, command.codeVerifier, now); err != nil {
			return invalidGrant(err)
		}
		auditIdentityID = &code.IdentityID

		session, err := sessiondomain.New(
			sessiondomain.ID(s.ids.New()),
			code.IdentityID,
			client.ID,
			code.AuthenticationLevel,
			code.AuthenticatedMethods,
			code.Scopes,
			nil,
			nil,
			nil, // todo Сделать и эти параметры
			code.AuthenticatedAt,
			now.Add(s.cfg.SessionIdleLifetime),
			now.Add(s.cfg.SessionLifetime),
			now,
		)
		if err != nil {
			return fmt.Errorf("create OAuth session: %w", err)
		}

		var refreshToken *string
		var family *refreshtokendomain.RefreshTokenFamily
		if client.AllowsGrantType(oauthclientdomain.GrantTypeRefreshToken) {
			raw, hash, err := s.tokenGenerator.New()
			if err != nil {
				return fmt.Errorf("generate refresh token: %w", err)
			}
			familyExpiresAt := now.Add(s.cfg.RefreshFamilyLifetime)
			family, err = refreshtokendomain.New(
				refreshtokendomain.ID(s.ids.New()),
				session.ID,
				refreshtokendomain.TokenID(s.ids.New()),
				refreshtokendomain.TokenHash(hash),
				domain.MinimumTime(now.Add(s.cfg.RefreshTokenLifetime), familyExpiresAt),
				familyExpiresAt,
				nil,
				nil,
				now,
			)
			if err != nil {
				return fmt.Errorf("create refresh token family: %w", err)
			}
			refreshToken = &raw
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
		identityClaims, err := s.scopedIdentityClaims(txCtx, code.IdentityID, code.Scopes)
		if err != nil {
			return fmt.Errorf("build OpenID Connect ID token claims: %w", err)
		}
		idToken, err := s.idTokens.Issue(txCtx, idtokenservice.IssueInput{
			IdentityID:           code.IdentityID,
			ClientID:             client.ClientID,
			Nonce:                code.Nonce,
			Identity:             toIDTokenIdentityClaims(identityClaims),
			AuthenticationLevel:  code.AuthenticationLevel,
			AuthenticatedMethods: code.AuthenticatedMethods,
			AuthenticatedAt:      code.AuthenticatedAt,
		})
		if err != nil {
			return fmt.Errorf("issue OpenID Connect ID token: %w", err)
		}
		if err := s.sessions.Create(txCtx, session); err != nil {
			return fmt.Errorf("save OAuth session: %w", err)
		}
		if family != nil {
			if err := s.refreshTokenFamilies.Create(txCtx, family); err != nil {
				return fmt.Errorf("save refresh token family: %w", err)
			}
		}

		scopes := make([]string, len(session.Scopes))
		for index := range session.Scopes {
			scopes[index] = string(session.Scopes[index])
		}
		sessionCreated, err := s.events.Create(sessiondomain.EventTypeCreated, sessiondomain.AggregateType, session.ID.UUID(), nil, sessiondomain.CreatedEvent{
			SessionID:            session.ID.UUID(),
			IdentityID:           session.IdentityID.UUID(),
			OAuthClientID:        session.OAuthClientID.UUID(),
			AuthenticationLevel:  session.AuthenticationLevel,
			AuthenticatedMethods: session.AuthenticatedMethods,
			Scopes:               session.Scopes,
			IPAddress:            session.IPAddress,
			UserAgent:            session.UserAgent,
			DeviceName:           session.DeviceName,
			AuthenticatedAt:      session.AuthenticatedAt,
			IdleExpiresAt:        session.IdleExpiresAt,
			AbsoluteExpiresAt:    session.AbsoluteExpiresAt,
			CreatedAt:            session.CreatedAt,
		}, now)
		if err != nil {
			return fmt.Errorf("create session created event: %w", err)
		}
		if err := s.outbox.Create(txCtx, sessionCreated); err != nil {
			return fmt.Errorf("save session created event to outbox: %w", err)
		}
		if family != nil {
			initialToken := family.Tokens[0]
			familyCreated, err := s.events.Create(refreshtokendomain.EventTypeCreated, refreshtokendomain.AggregateType, family.ID.UUID(), nil, refreshtokendomain.CreatedEvent{
				FamilyID:              family.ID.UUID(),
				SessionID:             family.SessionID.UUID(),
				InitialTokenID:        initialToken.ID.UUID(),
				InitialTokenExpiresAt: initialToken.ExpiresAt,
				AbsoluteExpiresAt:     family.AbsoluteExpiresAt,
				CreatedFromIP:         initialToken.CreatedFromIP,
				CreatedUserAgent:      initialToken.CreatedUserAgent,
				CreatedAt:             family.CreatedAt,
			}, now)
			if err != nil {
				return fmt.Errorf("create refresh token family created event: %w", err)
			}
			if err := s.outbox.Create(txCtx, familyCreated); err != nil {
				return fmt.Errorf("save refresh token family created event to outbox: %w", err)
			}
		}
		actor, err := securityeventdomain.NewIdentityActor(session.IdentityID)
		if err != nil {
			return fmt.Errorf("create authorization code exchange actor: %w", err)
		}
		if err := s.saveSecurityEvent(txCtx, securityEventInput{
			Type:          securityeventdomain.EventTypeAuthorizationCodeExchange,
			Result:        securityeventdomain.ResultSuccess,
			Actor:         actor,
			SessionID:     &session.ID,
			OAuthClientID: &client.ID,
			Metadata: map[string]any{
				"grant_type": "authorization_code",
				"scopes":     scopes,
			},
			OccurredAt: now,
		}); err != nil {
			return err
		}
		output = ExchangeAuthorizationCodeOutput{
			AccessToken:  accessToken.Value,
			TokenType:    "Bearer",
			ExpiresIn:    uint32(accessToken.ExpiresAt.Sub(accessToken.IssuedAt).Seconds()),
			RefreshToken: refreshToken,
			IDToken:      idToken.Value,
			Scope:        strings.Join(scopes, " "),
		}
		return nil
	})
	if err != nil {
		if !isOAuthFailure(err) {
			return ExchangeAuthorizationCodeOutput{}, err
		}
		actor := securityeventdomain.NewAnonymousActor()
		if auditIdentityID != nil {
			actor, _ = securityeventdomain.NewIdentityActor(*auditIdentityID)
		}
		if auditErr := s.saveSecurityEvent(ctx, securityEventInput{
			Type:          securityeventdomain.EventTypeAuthorizationCodeExchange,
			Result:        securityeventdomain.ResultFailure,
			Actor:         actor,
			OAuthClientID: auditClientID,
			Metadata: map[string]any{
				"grant_type": "authorization_code",
				"reason":     securityFailureReason(err),
			},
			OccurredAt: s.clock.Now(),
		}); auditErr != nil {
			return ExchangeAuthorizationCodeOutput{}, errors.Join(err, auditErr)
		}
		return ExchangeAuthorizationCodeOutput{}, err
	}

	return output, nil
}

func toIDTokenIdentityClaims(output GetUserInfoOutput) idtokenservice.IdentityClaims {
	claims := idtokenservice.IdentityClaims{
		EmailVerified:       output.EmailVerified,
		PhoneNumberVerified: output.PhoneNumberVerified,
		Name:                output.Name,
		FamilyName:          output.FamilyName,
		GivenName:           output.GivenName,
		MiddleName:          output.MiddleName,
		UpdatedAt:           output.UpdatedAt,
	}
	if output.Email != nil {
		value := string(*output.Email)
		claims.Email = &value
	}
	if output.PhoneNumber != nil {
		value := string(*output.PhoneNumber)
		claims.PhoneNumber = &value
	}
	return claims
}

func prepareExchangeAuthorizationCode(
	input ExchangeAuthorizationCodeInput,
) (exchangeAuthorizationCodeCommand, error) {
	clientID, err := oauthclientdomain.NewClientID(input.ClientID)
	if err != nil {
		return exchangeAuthorizationCodeCommand{}, err
	}
	if input.Code == "" {
		return exchangeAuthorizationCodeCommand{}, errors.New("authorization code is required")
	}
	redirectURI, err := oauthclientdomain.NewRedirectURI(input.RedirectURI)
	if err != nil {
		return exchangeAuthorizationCodeCommand{}, err
	}

	command := exchangeAuthorizationCodeCommand{
		clientID:     clientID,
		clientSecret: input.ClientSecret,
		code:         input.Code,
		redirectURI:  redirectURI,
	}
	if input.CodeVerifier != nil {
		verifier, err := oauthdomain.NewCodeVerifier(*input.CodeVerifier)
		if err != nil {
			return exchangeAuthorizationCodeCommand{}, err
		}
		command.codeVerifier = &verifier
	}

	return command, nil
}
