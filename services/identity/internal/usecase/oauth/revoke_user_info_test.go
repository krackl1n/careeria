package oauth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	identityprofiledomain "github.com/krackl1n/careeria/services/identity/internal/domain/identityprofile"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	refreshtokendomain "github.com/krackl1n/careeria/services/identity/internal/domain/refreshtokenfamily"
	securityeventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/securityevent"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
	accesstokenservice "github.com/krackl1n/careeria/services/identity/internal/service/accesstoken"
	"github.com/stretchr/testify/require"
)

func TestRevokeTokenRevokesRefreshTokenFamily(t *testing.T) {
	now := exchangeTestTime()
	client := newExchangeClient(t, now.Add(-time.Hour))
	session := newSessionWithScopes(t, client.ID, now, []oauthdomain.Scope{"openid"})
	family, err := refreshtokendomain.New(
		refreshtokendomain.ID(uuid.New()),
		session.ID,
		refreshtokendomain.TokenID(uuid.New()),
		"refresh-token-hash",
		now.Add(time.Hour),
		now.Add(24*time.Hour),
		nil,
		nil,
		now.Add(-time.Minute),
	)
	require.NoError(t, err)
	families := &revocationRefreshRepositoryStub{entity: family}
	service := &Service{
		oauthClients:         &oauthClientRepositoryStub{client: client},
		tokenGenerator:       &tokenGeneratorStub{},
		sessions:             &sessionRepositoryStub{entity: session},
		refreshTokenFamilies: families,
		tx:                   &transactionManagerStub{},
		clock:                clockStub{now: now},
	}
	setEventDependencies(service)
	clientID := string(client.ClientID)
	hint := "refresh_token"

	err = service.RevokeToken(context.Background(), RevokeTokenInput{
		ClientID:      clientID,
		Token:         "refresh-token",
		TokenTypeHint: &hint,
	})

	require.NoError(t, err)
	require.Same(t, family, families.updated)
	require.Equal(t, now, *family.RevokedAt)
	require.Equal(t, tokenRevokedReason, *family.RevokeReason)
	require.Equal(t, []eventdomain.Type{refreshtokendomain.EventTypeRevoked}, eventTypes(service.outbox.(*outboxWriterStub).messages))
	require.Equal(t, securityeventdomain.ResultSuccess, service.securityEvents.(*securityEventWriterStub).events[0].Result)
}

func TestRevokeTokenRevokesAccessTokenSession(t *testing.T) {
	now := exchangeTestTime()
	client := newExchangeClient(t, now.Add(-time.Hour))
	session := newSessionWithScopes(t, client.ID, now, []oauthdomain.Scope{"openid"})
	sessions := &sessionRepositoryStub{entity: session}
	service := &Service{
		oauthClients: &oauthClientRepositoryStub{client: client},
		accessTokenVerifier: accessTokenVerifierStub{token: accesstokenservice.VerifiedToken{
			ID:         uuid.New(),
			IdentityID: session.IdentityID,
			SessionID:  session.ID,
			ClientID:   client.ClientID,
			Scopes:     []oauthdomain.Scope{"openid"},
		}},
		sessions: sessions,
		tx:       &transactionManagerStub{},
		clock:    clockStub{now: now},
	}
	setEventDependencies(service)
	clientID := string(client.ClientID)
	hint := "access_token"

	err := service.RevokeToken(context.Background(), RevokeTokenInput{
		ClientID:      clientID,
		Token:         "access-token",
		TokenTypeHint: &hint,
	})

	require.NoError(t, err)
	require.Same(t, session, sessions.updated)
	require.Equal(t, now, *session.RevokedAt)
	require.Equal(t, tokenRevokedReason, *session.RevokeReason)
	require.Equal(t, []eventdomain.Type{sessiondomain.EventTypeRevoked}, eventTypes(service.outbox.(*outboxWriterStub).messages))
	require.Equal(t, securityeventdomain.ResultSuccess, service.securityEvents.(*securityEventWriterStub).events[0].Result)
}

func TestRevokeTokenIgnoresUnknownToken(t *testing.T) {
	now := exchangeTestTime()
	client := newExchangeClient(t, now.Add(-time.Hour))
	service := &Service{
		oauthClients:         &oauthClientRepositoryStub{client: client},
		tokenGenerator:       &tokenGeneratorStub{},
		accessTokenVerifier:  accessTokenVerifierStub{err: accesstokenservice.ErrInvalidToken},
		refreshTokenFamilies: &revocationRefreshRepositoryStub{err: refreshtokendomain.ErrTokenNotFound},
		tx:                   &transactionManagerStub{},
		clock:                clockStub{now: now},
	}
	setEventDependencies(service)

	err := service.RevokeToken(context.Background(), RevokeTokenInput{
		ClientID: string(client.ClientID),
		Token:    "unknown-token",
	})

	require.NoError(t, err)
	require.Empty(t, service.outbox.(*outboxWriterStub).messages)
	require.Equal(t, securityeventdomain.ResultFailure, service.securityEvents.(*securityEventWriterStub).events[0].Result)
}

func TestGetUserInfoReturnsClaimsAllowedByScopes(t *testing.T) {
	now := exchangeTestTime()
	client := newExchangeClient(t, now.Add(-time.Hour))
	identity := newUserInfoIdentity(t, now)
	scopes := []oauthdomain.Scope{"openid", "email", "phone", "profile"}
	session := newSessionForIdentity(t, identity.ID, client.ID, now, scopes)
	middleName := "Петрович"
	profile, err := identityprofiledomain.New(
		identity.ID,
		"Иванов",
		"Иван",
		&middleName,
		now.Add(-time.Minute),
	)
	require.NoError(t, err)
	service := &Service{
		accessTokenVerifier: accessTokenVerifierStub{token: accesstokenservice.VerifiedToken{
			ID:         uuid.New(),
			IdentityID: identity.ID,
			SessionID:  session.ID,
			ClientID:   client.ClientID,
			Scopes:     scopes,
		}},
		sessions:         &sessionRepositoryStub{entity: session},
		identities:       identityRepositoryStub{entity: identity},
		identityProfiles: identityProfileRepositoryStub{entity: profile},
		tx:               &transactionManagerStub{},
		clock:            clockStub{now: now},
	}
	setEventDependencies(service)

	output, err := service.GetUserInfo(context.Background(), GetUserInfoInput{AccessToken: "access-token"})

	require.NoError(t, err)
	require.Equal(t, identity.ID, output.Subject)
	require.Equal(t, identitydomain.EmailAddress("user@example.com"), *output.Email)
	require.True(t, *output.EmailVerified)
	require.Equal(t, identitydomain.PhoneNumber("+79991234567"), *output.PhoneNumber)
	require.True(t, *output.PhoneNumberVerified)
	require.Equal(t, "Иван Петрович Иванов", *output.Name)
	require.Equal(t, "Иванов", *output.FamilyName)
	require.Equal(t, "Иван", *output.GivenName)
	require.Equal(t, "Петрович", *output.MiddleName)
	require.Equal(t, profile.UpdatedAt, *output.UpdatedAt)
	require.Equal(t, securityeventdomain.ResultSuccess, service.securityEvents.(*securityEventWriterStub).events[0].Result)
}

func TestGetUserInfoRejectsInvalidAccessToken(t *testing.T) {
	service := &Service{
		accessTokenVerifier: accessTokenVerifierStub{err: accesstokenservice.ErrInvalidToken},
	}

	_, err := service.GetUserInfo(context.Background(), GetUserInfoInput{AccessToken: "invalid-token"})

	require.Error(t, err)
	require.True(t, apperr.IsCode(err, "oauth_invalid_token"))
}

type accessTokenVerifierStub struct {
	token accesstokenservice.VerifiedToken
	err   error
}

func (s accessTokenVerifierStub) Verify(context.Context, string) (accesstokenservice.VerifiedToken, error) {
	return s.token, s.err
}

type revocationRefreshRepositoryStub struct {
	entity  *refreshtokendomain.RefreshTokenFamily
	err     error
	updated *refreshtokendomain.RefreshTokenFamily
}

func (*revocationRefreshRepositoryStub) Create(context.Context, *refreshtokendomain.RefreshTokenFamily) error {
	return nil
}

func (s *revocationRefreshRepositoryStub) GetByTokenHashForUpdate(context.Context, refreshtokendomain.TokenHash) (*refreshtokendomain.RefreshTokenFamily, error) {
	return s.entity, s.err
}

func (s *revocationRefreshRepositoryStub) Update(_ context.Context, entity *refreshtokendomain.RefreshTokenFamily) error {
	s.updated = entity
	return nil
}

type identityRepositoryStub struct {
	entity *identitydomain.Identity
	err    error
}

func (s identityRepositoryStub) Get(context.Context, identitydomain.ID) (*identitydomain.Identity, error) {
	return s.entity, s.err
}

type identityProfileRepositoryStub struct {
	entity *identityprofiledomain.Profile
	err    error
}

func (s identityProfileRepositoryStub) Get(context.Context, identitydomain.ID) (*identityprofiledomain.Profile, error) {
	return s.entity, s.err
}

func newSessionWithScopes(
	t *testing.T,
	clientID oauthclientdomain.ID,
	now time.Time,
	scopes []oauthdomain.Scope,
) *sessiondomain.Session {
	return newSessionForIdentity(t, identitydomain.ID(uuid.New()), clientID, now, scopes)
}

func newSessionForIdentity(
	t *testing.T,
	identityID identitydomain.ID,
	clientID oauthclientdomain.ID,
	now time.Time,
	scopes []oauthdomain.Scope,
) *sessiondomain.Session {
	t.Helper()
	session, err := sessiondomain.New(
		sessiondomain.ID(uuid.New()),
		identityID,
		clientID,
		authenticationdomain.LevelSingleFactor,
		[]authenticationdomain.Method{authenticationdomain.MethodPassword},
		scopes,
		nil,
		nil,
		nil,
		now.Add(-time.Hour),
		now.Add(time.Hour),
		now.Add(24*time.Hour),
		now.Add(-time.Minute),
	)
	require.NoError(t, err)
	return session
}

func newUserInfoIdentity(t *testing.T, now time.Time) *identitydomain.Identity {
	return newUserInfoIdentityWithID(t, identitydomain.ID(uuid.New()), now)
}

func newUserInfoIdentityWithID(
	t *testing.T,
	identityID identitydomain.ID,
	now time.Time,
) *identitydomain.Identity {
	t.Helper()
	email, err := identitydomain.NewEmailAddress("user@example.com")
	require.NoError(t, err)
	identity, err := identitydomain.NewWithEmail(
		identityID,
		identitydomain.EmailID(uuid.New()),
		email,
		now.Add(-10*time.Minute),
	)
	require.NoError(t, err)
	require.NoError(t, identity.VerifyEmail(identity.Emails[0].ID, now.Add(-9*time.Minute)))
	require.NoError(t, identity.Activate(now.Add(-8*time.Minute)))
	phone, err := identitydomain.NewPhoneNumber("+79991234567")
	require.NoError(t, err)
	require.NoError(t, identity.AddPhone(identitydomain.PhoneID(uuid.New()), phone, now.Add(-7*time.Minute)))
	require.NoError(t, identity.VerifyPhone(identity.Phones[0].ID, now.Add(-6*time.Minute)))
	return identity
}
