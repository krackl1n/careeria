package oauth

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	identityprofiledomain "github.com/krackl1n/careeria/services/identity/internal/domain/identityprofile"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	securityeventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/securityevent"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
	accesstokenservice "github.com/krackl1n/careeria/services/identity/internal/service/accesstoken"
)

type GetUserInfoInput struct {
	AccessToken string
}

type GetUserInfoOutput struct {
	Subject identitydomain.ID

	Email         *identitydomain.EmailAddress
	EmailVerified *bool

	PhoneNumber         *identitydomain.PhoneNumber
	PhoneNumberVerified *bool

	Name       *string
	FamilyName *string
	GivenName  *string
	MiddleName *string
	UpdatedAt  *time.Time
}

func (s *Service) GetUserInfo(ctx context.Context, input GetUserInfoInput) (GetUserInfoOutput, error) {
	if input.AccessToken == "" {
		return GetUserInfoOutput{}, invalidAccessToken(errors.New("access token is required"))
	}

	token, err := s.accessTokenVerifier.Verify(ctx, input.AccessToken)
	if err != nil {
		if errors.Is(err, accesstokenservice.ErrInvalidToken) {
			return GetUserInfoOutput{}, invalidAccessToken(err)
		}
		return GetUserInfoOutput{}, fmt.Errorf("verify UserInfo access token: %w", err)
	}
	now := s.clock.Now()
	var output GetUserInfoOutput
	var auditClientID *oauthclientdomain.ID
	err = s.tx.Do(ctx, func(txCtx context.Context) error {
		session, err := s.sessions.Get(txCtx, token.SessionID)
		if err != nil {
			if errors.Is(err, sessiondomain.ErrNotFound) {
				return invalidAccessToken(err)
			}
			return fmt.Errorf("get UserInfo session: %w", err)
		}
		auditClientID = &session.OAuthClientID
		if session.IdentityID != token.IdentityID || !scopesBelongToSession(token.Scopes, session.Scopes) {
			return invalidAccessToken(errors.New("access token claims do not match session"))
		}
		if err := session.ValidateForUse(now); err != nil {
			return invalidAccessToken(err)
		}
		if !slices.Contains(token.Scopes, oauthdomain.Scope("openid")) {
			return insufficientScope(errors.New("openid scope is required"))
		}

		output, err = s.scopedIdentityClaims(txCtx, token.IdentityID, token.Scopes)
		if err != nil {
			if errors.Is(err, identitydomain.ErrNotFound) || errors.Is(err, identitydomain.ErrInvalidStatus) {
				return invalidAccessToken(err)
			}
			return err
		}

		actor, err := securityeventdomain.NewIdentityActor(token.IdentityID)
		if err != nil {
			return fmt.Errorf("create UserInfo actor: %w", err)
		}
		return s.saveSecurityEvent(txCtx, securityEventInput{
			Type:          securityeventdomain.EventTypeUserInfoAccess,
			Result:        securityeventdomain.ResultSuccess,
			Actor:         actor,
			SessionID:     &session.ID,
			OAuthClientID: &session.OAuthClientID,
			Metadata: map[string]any{
				"scopes": token.Scopes,
			},
			OccurredAt: now,
		})
	})
	if err != nil {
		if !isOAuthFailure(err) {
			return GetUserInfoOutput{}, err
		}
		actor, actorErr := securityeventdomain.NewIdentityActor(token.IdentityID)
		if actorErr != nil {
			return GetUserInfoOutput{}, errors.Join(err, actorErr)
		}
		if auditErr := s.saveSecurityEvent(ctx, securityEventInput{
			Type:          securityeventdomain.EventTypeUserInfoAccess,
			Result:        securityeventdomain.ResultFailure,
			Actor:         actor,
			SessionID:     &token.SessionID,
			OAuthClientID: auditClientID,
			Metadata: map[string]any{
				"reason": securityFailureReason(err),
			},
			OccurredAt: s.clock.Now(),
		}); auditErr != nil {
			return GetUserInfoOutput{}, errors.Join(err, auditErr)
		}
		return GetUserInfoOutput{}, err
	}

	return output, nil
}

func (s *Service) scopedIdentityClaims(
	ctx context.Context,
	identityID identitydomain.ID,
	scopes []oauthdomain.Scope,
) (GetUserInfoOutput, error) {
	identity, err := s.identities.Get(ctx, identityID)
	if err != nil {
		return GetUserInfoOutput{}, fmt.Errorf("get scoped identity claims: %w", err)
	}
	if identity.Status != identitydomain.StatusActive {
		return GetUserInfoOutput{}, identitydomain.ErrInvalidStatus
	}

	output := GetUserInfoOutput{Subject: identity.ID}
	if slices.Contains(scopes, oauthdomain.Scope("email")) {
		setUserInfoEmail(&output, identity)
	}
	if slices.Contains(scopes, oauthdomain.Scope("phone")) {
		setUserInfoPhone(&output, identity)
	}
	if slices.Contains(scopes, oauthdomain.Scope("profile")) {
		if err := s.setUserInfoProfile(ctx, &output, identity); err != nil {
			return GetUserInfoOutput{}, err
		}
	}

	return output, nil
}

func scopesBelongToSession(tokenScopes, sessionScopes []oauthdomain.Scope) bool {
	for _, scope := range tokenScopes {
		if !slices.Contains(sessionScopes, scope) {
			return false
		}
	}
	return true
}

func setUserInfoEmail(output *GetUserInfoOutput, identity *identitydomain.Identity) {
	for index := range identity.Emails {
		email := &identity.Emails[index]
		if !email.IsPrimaryEmail() {
			continue
		}
		address := email.Address
		verified := email.VerifiedAt != nil
		output.Email = &address
		output.EmailVerified = &verified
		return
	}
}

func setUserInfoPhone(output *GetUserInfoOutput, identity *identitydomain.Identity) {
	for index := range identity.Phones {
		phone := &identity.Phones[index]
		if !phone.IsPrimaryPhone() {
			continue
		}
		number := phone.Number
		verified := phone.VerifiedAt != nil
		output.PhoneNumber = &number
		output.PhoneNumberVerified = &verified
		return
	}
}

func (s *Service) setUserInfoProfile(
	ctx context.Context,
	output *GetUserInfoOutput,
	identity *identitydomain.Identity,
) error {
	updatedAt := identity.UpdatedAt
	output.UpdatedAt = &updatedAt

	profile, err := s.identityProfiles.Get(ctx, identity.ID)
	if err != nil {
		if errors.Is(err, identityprofiledomain.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("get UserInfo profile: %w", err)
	}
	if profile.UpdatedAt.After(updatedAt) {
		updatedAt = profile.UpdatedAt
		output.UpdatedAt = &updatedAt
	}

	familyName := profile.FamilyName
	givenName := profile.GivenName
	output.FamilyName = &familyName
	output.GivenName = &givenName
	if profile.MiddleName != nil {
		middleName := *profile.MiddleName
		output.MiddleName = &middleName
	}

	nameParts := []string{profile.GivenName}
	if profile.MiddleName != nil {
		nameParts = append(nameParts, *profile.MiddleName)
	}
	nameParts = append(nameParts, profile.FamilyName)
	name := strings.Join(nameParts, " ")
	output.Name = &name

	return nil
}
