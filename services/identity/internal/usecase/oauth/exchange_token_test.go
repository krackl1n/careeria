package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/apperr"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	authorizationcodedomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationcode"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	identityprofiledomain "github.com/krackl1n/careeria/services/identity/internal/domain/identityprofile"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	refreshtokendomain "github.com/krackl1n/careeria/services/identity/internal/domain/refreshtokenfamily"
	securityeventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/securityevent"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
	accesstokenservice "github.com/krackl1n/careeria/services/identity/internal/service/accesstoken"
	idtokenservice "github.com/krackl1n/careeria/services/identity/internal/service/idtoken"
	"github.com/stretchr/testify/require"
)

const exchangeVerifier = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"

func TestExchangeAuthorizationCodeCreatesSessionAndRefreshFamily(t *testing.T) {
	now := exchangeTestTime()
	client := newExchangeClient(t, now.Add(-time.Hour))
	code := newExchangeAuthorizationCode(t, client.ID, now)
	sessions := &sessionRepositoryStub{}
	families := &refreshFamilyRepositoryStub{}
	accessTokens := &accessTokenIssuerStub{issued: issuedAccessToken(now)}
	service := newExchangeService(
		t,
		now,
		client,
		&authorizationCodeRepositoryStub{code: code},
		&tokenGeneratorStub{nextRaw: "refresh-token", nextHash: "refresh-hash"},
		sessions,
		families,
		accessTokens,
	)
	clientID := string(client.ClientID)
	redirectURI := string(code.RedirectURI)
	verifier := exchangeVerifier

	output, err := service.ExchangeAuthorizationCode(context.Background(), ExchangeAuthorizationCodeInput{
		ClientID:     clientID,
		Code:         "authorization-code",
		RedirectURI:  redirectURI,
		CodeVerifier: &verifier,
	})

	require.NoError(t, err)
	require.Equal(t, "access-token", output.AccessToken)
	require.Equal(t, "Bearer", output.TokenType)
	require.Equal(t, uint32(900), output.ExpiresIn)
	require.Equal(t, "refresh-token", *output.RefreshToken)
	require.Equal(t, "id-token", output.IDToken)
	require.Equal(t, "openid profile email phone", output.Scope)
	require.NotNil(t, sessions.created)
	require.Equal(t, code.AuthenticatedAt, sessions.created.AuthenticatedAt)
	require.Equal(t, now.Add(exchangeConfig().SessionIdleLifetime), sessions.created.IdleExpiresAt)
	require.NotNil(t, families.created)
	require.Equal(t, sessions.created.ID, families.created.SessionID)
	require.Equal(t, refreshtokendomain.TokenHash("refresh-hash"), families.created.Tokens[0].Hash)
	require.Equal(t, sessions.created.ID, accessTokens.input.SessionID)
	require.Equal(t, code.AuthenticatedAt, accessTokens.input.AuthenticatedAt)
	idTokens := service.idTokens.(*idTokenIssuerStub)
	require.Equal(t, "Иван Петрович Иванов", *idTokens.input.Identity.Name)
	require.Equal(t, "user@example.com", *idTokens.input.Identity.Email)
	require.True(t, *idTokens.input.Identity.EmailVerified)
	require.Equal(t, "+79991234567", *idTokens.input.Identity.PhoneNumber)
	require.True(t, *idTokens.input.Identity.PhoneNumberVerified)
	require.Equal(t, []eventdomain.Type{
		sessiondomain.EventTypeCreated,
		refreshtokendomain.EventTypeCreated,
	}, eventTypes(service.outbox.(*outboxWriterStub).messages))
	require.Equal(t, securityeventdomain.ResultSuccess, service.securityEvents.(*securityEventWriterStub).events[0].Result)
}

func TestExchangeRefreshTokenRotatesTokenAndUpdatesSession(t *testing.T) {
	createdAt := exchangeTestTime().Add(-time.Hour)
	now := exchangeTestTime()
	client := newExchangeClient(t, createdAt)
	session := newExchangeSession(t, client.ID, createdAt)
	family, err := refreshtokendomain.New(
		refreshtokendomain.ID(uuid.New()),
		session.ID,
		refreshtokendomain.TokenID(uuid.New()),
		"presented-hash",
		now.Add(time.Hour),
		now.Add(24*time.Hour),
		nil,
		nil,
		createdAt,
	)
	require.NoError(t, err)
	sessions := &sessionRepositoryStub{entity: session}
	families := &refreshFamilyRepositoryStub{entity: family}
	accessTokens := &accessTokenIssuerStub{issued: issuedAccessToken(now)}
	service := newExchangeService(
		t,
		now,
		client,
		&authorizationCodeRepositoryStub{},
		&tokenGeneratorStub{
			nextRaw:  "next-refresh-token",
			nextHash: "next-hash",
			hashes:   map[string]string{"presented-token": "presented-hash"},
		},
		sessions,
		families,
		accessTokens,
	)
	clientID := string(client.ClientID)

	output, err := service.RefreshAccessToken(context.Background(), RefreshAccessTokenInput{
		ClientID:     clientID,
		RefreshToken: "presented-token",
	})

	require.NoError(t, err)
	require.Equal(t, "next-refresh-token", output.RefreshToken)
	require.Len(t, families.updated.Tokens, 2)
	require.Equal(t, refreshtokendomain.TokenHash("next-hash"), families.updated.Tokens[1].Hash)
	require.Equal(t, now, *sessions.updated.LastSeenAt)
	require.Nil(t, sessions.updated.RevokedAt)
	require.Equal(t, 1, accessTokens.calls)
	require.Equal(t, []eventdomain.Type{refreshtokendomain.EventTypeTokenRotated}, eventTypes(service.outbox.(*outboxWriterStub).messages))
	require.Equal(t, securityeventdomain.ResultSuccess, service.securityEvents.(*securityEventWriterStub).events[0].Result)
}

func TestRefreshTokenReuseCommitsFamilyAndSessionRevocation(t *testing.T) {
	createdAt := exchangeTestTime().Add(-time.Hour)
	rotatedAt := exchangeTestTime().Add(-time.Minute)
	now := exchangeTestTime()
	client := newExchangeClient(t, createdAt)
	session := newExchangeSession(t, client.ID, createdAt)
	presentedTokenID := refreshtokendomain.TokenID(uuid.New())
	family, err := refreshtokendomain.New(
		refreshtokendomain.ID(uuid.New()),
		session.ID,
		presentedTokenID,
		"presented-hash",
		now.Add(time.Hour),
		now.Add(24*time.Hour),
		nil,
		nil,
		createdAt,
	)
	require.NoError(t, err)
	_, err = family.Rotate(
		presentedTokenID,
		refreshtokendomain.TokenID(uuid.New()),
		"current-hash",
		now.Add(time.Hour),
		nil,
		nil,
		rotatedAt,
	)
	require.NoError(t, err)
	tx := &transactionManagerStub{}
	sessions := &sessionRepositoryStub{entity: session}
	families := &refreshFamilyRepositoryStub{entity: family}
	accessTokens := &accessTokenIssuerStub{issued: issuedAccessToken(now)}
	service := newExchangeService(
		t,
		now,
		client,
		&authorizationCodeRepositoryStub{},
		&tokenGeneratorStub{
			nextRaw:  "discarded-token",
			nextHash: "discarded-hash",
			hashes:   map[string]string{"reused-token": "presented-hash"},
		},
		sessions,
		families,
		accessTokens,
	)
	service.tx = tx
	clientID := string(client.ClientID)

	_, err = service.RefreshAccessToken(context.Background(), RefreshAccessTokenInput{
		ClientID:     clientID,
		RefreshToken: "reused-token",
	})

	require.Error(t, err)
	require.True(t, apperr.IsCode(err, "oauth_invalid_grant"))
	require.NoError(t, tx.callbackError)
	require.NotNil(t, families.updated.ReuseDetectedAt)
	require.NotNil(t, families.updated.RevokedAt)
	require.NotNil(t, sessions.updated.RevokedAt)
	require.Equal(t, 0, accessTokens.calls)
	require.Equal(t, []eventdomain.Type{
		refreshtokendomain.EventTypeReuseDetected,
		sessiondomain.EventTypeRevoked,
	}, eventTypes(service.outbox.(*outboxWriterStub).messages))
	require.Equal(t, securityeventdomain.ResultBlocked, service.securityEvents.(*securityEventWriterStub).events[0].Result)
}

func TestTokenExchangeAuthenticatesConfidentialClient(t *testing.T) {
	createdAt := exchangeTestTime().Add(-time.Hour)
	now := exchangeTestTime()
	clientID, err := oauthclientdomain.NewClientID("server-client")
	require.NoError(t, err)
	redirectURI, err := oauthclientdomain.NewRedirectURI("https://client.example/callback")
	require.NoError(t, err)
	client, err := oauthclientdomain.NewConfidential(
		oauthclientdomain.ID(uuid.New()),
		clientID,
		"Server client",
		nil,
		nil,
		oauthclientdomain.RedirectURIID(uuid.New()),
		redirectURI,
		oauthclientdomain.CredentialID(uuid.New()),
		oauthclientdomain.SecretHash("encoded-secret"),
		nil,
		nil,
		createdAt,
	)
	require.NoError(t, err)
	repository := &oauthClientRepositoryStub{client: client}
	service := &Service{
		oauthClients:  repository,
		clientSecrets: clientSecretVerifierStub{},
	}
	secret := "client-secret"

	authenticated, err := service.getAndAuthenticateClient(
		context.Background(),
		clientID,
		&secret,
		now,
	)

	require.NoError(t, err)
	require.Same(t, client, authenticated)
	require.Same(t, client, repository.updated)
	require.Equal(t, now, *client.Credentials[0].LastUsedAt)
}

func newExchangeService(
	t *testing.T,
	now time.Time,
	client *oauthclientdomain.OAuthClient,
	codes *authorizationCodeRepositoryStub,
	tokens *tokenGeneratorStub,
	sessions *sessionRepositoryStub,
	families *refreshFamilyRepositoryStub,
	accessTokens *accessTokenIssuerStub,
) *Service {
	t.Helper()
	identityID := identitydomain.ID(uuid.New())
	if codes.code != nil {
		identityID = codes.code.IdentityID
	}
	identity := newUserInfoIdentityWithID(t, identityID, now)
	middleName := "Петрович"
	profile, err := identityprofiledomain.New(identity.ID, "Иванов", "Иван", &middleName, now.Add(-time.Minute))
	require.NoError(t, err)
	service := &Service{
		cfg:                  exchangeConfig(),
		oauthClients:         &oauthClientRepositoryStub{client: client},
		tokenGenerator:       tokens,
		authorizationCode:    codes,
		sessions:             sessions,
		refreshTokenFamilies: families,
		accessTokens:         accessTokens,
		idTokens:             &idTokenIssuerStub{},
		identities:           identityRepositoryStub{entity: identity},
		identityProfiles:     identityProfileRepositoryStub{entity: profile},
		tx:                   &transactionManagerStub{},
		ids:                  &idGeneratorStub{},
		clock:                clockStub{now: now},
	}
	setEventDependencies(service)
	return service
}

func exchangeConfig() Config {
	return Config{
		AuthorizationLifetime: 10 * time.Minute,
		CodeLifetime:          2 * time.Minute,
		SessionIdleLifetime:   24 * time.Hour,
		SessionLifetime:       30 * 24 * time.Hour,
		RefreshTokenLifetime:  24 * time.Hour,
		RefreshFamilyLifetime: 30 * 24 * time.Hour,
	}
}

func newExchangeClient(t *testing.T, createdAt time.Time) *oauthclientdomain.OAuthClient {
	t.Helper()
	clientID, err := oauthclientdomain.NewClientID("web-client")
	require.NoError(t, err)
	redirectURI, err := oauthclientdomain.NewRedirectURI("https://client.example/callback")
	require.NoError(t, err)
	client, err := oauthclientdomain.NewPublic(
		oauthclientdomain.ID(uuid.New()),
		clientID,
		"Web client",
		nil,
		nil,
		oauthclientdomain.RedirectURIID(uuid.New()),
		redirectURI,
		createdAt,
	)
	require.NoError(t, err)
	require.NoError(t, client.AddGrantType(oauthclientdomain.GrantTypeRefreshToken, createdAt))
	return client
}

func newExchangeAuthorizationCode(
	t *testing.T,
	clientID oauthclientdomain.ID,
	now time.Time,
) *authorizationcodedomain.AuthorizationCode {
	t.Helper()
	digest := sha256.Sum256([]byte(exchangeVerifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])
	method := "S256"
	pkce, err := oauthdomain.NewPKCE(&challenge, &method)
	require.NoError(t, err)
	redirectURI, err := oauthclientdomain.NewRedirectURI("https://client.example/callback")
	require.NoError(t, err)
	code, err := authorizationcodedomain.New(
		"authorization-code-hash",
		clientID,
		redirectURI,
		identitydomain.ID(uuid.New()),
		authenticationdomain.LevelSingleFactor,
		[]authenticationdomain.Method{authenticationdomain.MethodPassword},
		[]oauthdomain.Scope{"openid", "profile", "email", "phone"},
		nil,
		pkce,
		now.Add(-5*time.Minute),
		now.Add(time.Minute),
		now.Add(-time.Minute),
	)
	require.NoError(t, err)
	return code
}

func newExchangeSession(
	t *testing.T,
	clientID oauthclientdomain.ID,
	createdAt time.Time,
) *sessiondomain.Session {
	t.Helper()
	session, err := sessiondomain.New(
		sessiondomain.ID(uuid.New()),
		identitydomain.ID(uuid.New()),
		clientID,
		authenticationdomain.LevelSingleFactor,
		[]authenticationdomain.Method{authenticationdomain.MethodPassword},
		[]oauthdomain.Scope{"openid", "profile"},
		nil,
		nil,
		nil,
		createdAt,
		createdAt.Add(24*time.Hour),
		createdAt.Add(30*24*time.Hour),
		createdAt,
	)
	require.NoError(t, err)
	return session
}

func issuedAccessToken(now time.Time) accesstokenservice.IssuedToken {
	return accesstokenservice.IssuedToken{
		Value:     "access-token",
		ID:        uuid.New(),
		IssuedAt:  now,
		ExpiresAt: now.Add(15 * time.Minute),
	}
}

func exchangeTestTime() time.Time {
	return time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
}

type oauthClientRepositoryStub struct {
	client  *oauthclientdomain.OAuthClient
	updated *oauthclientdomain.OAuthClient
}

func (s *oauthClientRepositoryStub) GetByClientID(context.Context, oauthclientdomain.ClientID) (*oauthclientdomain.OAuthClient, error) {
	return s.client, nil
}

func (s *oauthClientRepositoryStub) Update(_ context.Context, client *oauthclientdomain.OAuthClient) error {
	s.updated = client
	return nil
}

type clientSecretVerifierStub struct{}

func (clientSecretVerifierStub) Verify(secret, encodedHash string) (bool, error) {
	return secret == "client-secret" && encodedHash == "encoded-secret", nil
}

type authorizationCodeRepositoryStub struct {
	code *authorizationcodedomain.AuthorizationCode
}

func (*authorizationCodeRepositoryStub) Create(context.Context, *authorizationcodedomain.AuthorizationCode) error {
	return nil
}

func (s *authorizationCodeRepositoryStub) Consume(context.Context, authorizationcodedomain.Hash) (*authorizationcodedomain.AuthorizationCode, error) {
	return s.code, nil
}

type tokenGeneratorStub struct {
	nextRaw  string
	nextHash string
	hashes   map[string]string
}

func (s *tokenGeneratorStub) New() (string, string, error) {
	return s.nextRaw, s.nextHash, nil
}

func (s *tokenGeneratorStub) Hash(raw string) string {
	if hash, ok := s.hashes[raw]; ok {
		return hash
	}
	return raw + "-hash"
}

type sessionRepositoryStub struct {
	entity  *sessiondomain.Session
	created *sessiondomain.Session
	updated *sessiondomain.Session
}

func (s *sessionRepositoryStub) Create(_ context.Context, entity *sessiondomain.Session) error {
	s.created = entity
	return nil
}

func (s *sessionRepositoryStub) Get(context.Context, sessiondomain.ID) (*sessiondomain.Session, error) {
	return s.entity, nil
}

func (s *sessionRepositoryStub) Update(_ context.Context, entity *sessiondomain.Session) error {
	s.updated = entity
	return nil
}

type refreshFamilyRepositoryStub struct {
	entity  *refreshtokendomain.RefreshTokenFamily
	created *refreshtokendomain.RefreshTokenFamily
	updated *refreshtokendomain.RefreshTokenFamily
}

func (s *refreshFamilyRepositoryStub) Create(_ context.Context, entity *refreshtokendomain.RefreshTokenFamily) error {
	s.created = entity
	return nil
}

func (s *refreshFamilyRepositoryStub) GetByTokenHashForUpdate(context.Context, refreshtokendomain.TokenHash) (*refreshtokendomain.RefreshTokenFamily, error) {
	return s.entity, nil
}

func (s *refreshFamilyRepositoryStub) Update(_ context.Context, entity *refreshtokendomain.RefreshTokenFamily) error {
	s.updated = entity
	return nil
}

type accessTokenIssuerStub struct {
	issued accesstokenservice.IssuedToken
	input  accesstokenservice.IssueInput
	calls  int
}

type idTokenIssuerStub struct {
	input idtokenservice.IssueInput
}

func (s *idTokenIssuerStub) Issue(_ context.Context, input idtokenservice.IssueInput) (idtokenservice.IssuedToken, error) {
	s.input = input
	return idtokenservice.IssuedToken{Value: "id-token"}, nil
}

func (s *accessTokenIssuerStub) Issue(_ context.Context, input accesstokenservice.IssueInput) (accesstokenservice.IssuedToken, error) {
	s.calls++
	s.input = input
	return s.issued, nil
}

type transactionManagerStub struct {
	callbackError error
}

func (s *transactionManagerStub) Do(ctx context.Context, fn func(context.Context) error) error {
	s.callbackError = fn(ctx)
	return s.callbackError
}

type idGeneratorStub struct{}

func (*idGeneratorStub) New() uuid.UUID {
	return uuid.New()
}

type clockStub struct {
	now time.Time
}

func (s clockStub) Now() time.Time {
	return s.now
}

type outboxWriterStub struct {
	messages []eventdomain.Message
}

func (s *outboxWriterStub) Create(_ context.Context, message eventdomain.Message) error {
	s.messages = append(s.messages, message)
	return nil
}

type securityEventWriterStub struct {
	events []*securityeventdomain.SecurityEvent
}

func (s *securityEventWriterStub) Create(_ context.Context, event *securityeventdomain.SecurityEvent) error {
	s.events = append(s.events, event)
	return nil
}

func setEventDependencies(service *Service) {
	if service.ids == nil {
		service.ids = &idGeneratorStub{}
	}
	service.events = eventdomain.NewFactory(service.ids)
	service.outbox = &outboxWriterStub{}
	service.securityEvents = &securityEventWriterStub{}
}

func eventTypes(messages []eventdomain.Message) []eventdomain.Type {
	types := make([]eventdomain.Type, len(messages))
	for index := range messages {
		types[index] = messages[index].EventType
	}
	return types
}
