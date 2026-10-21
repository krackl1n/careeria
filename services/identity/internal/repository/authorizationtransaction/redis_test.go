package authorizationtransaction

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	openidconnectdomain "github.com/krackl1n/careeria/services/identity/internal/domain/openidconnect"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRedisRepositoryCreate(t *testing.T) {
	t.Parallel()

	entity := transactionFixture()
	var command []any
	client := &redisClientStub{
		doFn: func(_ context.Context, args ...any) *goredis.Cmd {
			command = args
			return goredis.NewCmdResult("OK", nil)
		},
	}

	err := NewRedisRepository(client).Create(context.Background(), entity)

	require.NoError(t, err)
	require.Len(t, command, 6)
	require.Equal(t, "SET", command[0])
	require.Equal(t, key(entity.ID), command[1])
	require.Equal(t, "PXAT", command[3])
	require.Equal(t, entity.ExpiresAt.UnixMilli(), command[4])
	require.Equal(t, "NX", command[5])

	payload, ok := command[2].([]byte)
	require.True(t, ok)
	decoded, err := decode(payload)
	require.NoError(t, err)
	require.Equal(t, entity, decoded)
}

func TestRedisRepositoryCreateErrors(t *testing.T) {
	t.Parallel()

	entity := transactionFixture()
	backendErr := errors.New("redis unavailable")

	tests := []struct {
		name    string
		entity  *domain.AuthorizationTransaction
		result  *goredis.Cmd
		wantErr error
	}{
		{
			name:    "nil entity",
			wantErr: errors.New("create authorization transaction: entity is nil"),
		},
		{
			name:    "already exists",
			entity:  entity,
			result:  goredis.NewCmdResult(nil, goredis.Nil),
			wantErr: domain.ErrAlreadyExists,
		},
		{
			name:    "backend error",
			entity:  entity,
			result:  goredis.NewCmdResult(nil, backendErr),
			wantErr: backendErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &redisClientStub{
				doFn: func(context.Context, ...any) *goredis.Cmd {
					return tt.result
				},
			}

			err := NewRedisRepository(client).Create(context.Background(), tt.entity)

			if tt.name == "nil entity" {
				require.EqualError(t, err, tt.wantErr.Error())
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestRedisRepositoryGet(t *testing.T) {
	t.Parallel()

	entity := transactionFixture()
	payload, err := encode(entity)
	require.NoError(t, err)
	client := &redisClientStub{
		getFn: func(_ context.Context, gotKey string) *goredis.StringCmd {
			require.Equal(t, key(entity.ID), gotKey)
			return goredis.NewStringResult(string(payload), nil)
		},
	}

	got, err := NewRedisRepository(client).Get(context.Background(), entity.ID)

	require.NoError(t, err)
	require.Equal(t, entity, got)
}

func TestRedisRepositoryGetErrors(t *testing.T) {
	t.Parallel()

	id := transactionFixture().ID
	backendErr := errors.New("redis unavailable")
	mismatched := transactionFixture()
	mismatched.ID = domain.ID(uuid.New())
	mismatchedPayload, err := encode(mismatched)
	require.NoError(t, err)

	tests := []struct {
		name    string
		result  *goredis.StringCmd
		wantErr error
	}{
		{
			name:    "not found",
			result:  goredis.NewStringResult("", goredis.Nil),
			wantErr: domain.ErrNotFound,
		},
		{
			name:    "backend error",
			result:  goredis.NewStringResult("", backendErr),
			wantErr: backendErr,
		},
		{
			name:    "invalid JSON",
			result:  goredis.NewStringResult("{", nil),
			wantErr: errors.New("decode authorization transaction"),
		},
		{
			name: "unsupported model",
			result: goredis.NewStringResult(
				`{"version":2}`,
				nil,
			),
			wantErr: errors.New("unsupported model version 2"),
		},
		{
			name:    "identifier does not match key",
			result:  goredis.NewStringResult(string(mismatchedPayload), nil),
			wantErr: errors.New("stored identifier does not match key"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &redisClientStub{
				getFn: func(context.Context, string) *goredis.StringCmd {
					return tt.result
				},
			}

			_, err := NewRedisRepository(client).Get(context.Background(), id)

			if tt.name == "invalid JSON" ||
				tt.name == "unsupported model" ||
				tt.name == "identifier does not match key" {
				require.ErrorContains(t, err, tt.wantErr.Error())
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestRedisRepositoryUpdate(t *testing.T) {
	t.Parallel()

	entity := transactionFixture()
	entity.Status = domain.StatusReady
	entity.NextStep = domain.NextStepReady

	var command []any
	client := &redisClientStub{
		doFn: func(_ context.Context, args ...any) *goredis.Cmd {
			command = args
			return goredis.NewCmdResult("OK", nil)
		},
	}

	err := NewRedisRepository(client).Update(context.Background(), entity)

	require.NoError(t, err)
	require.Equal(t, "XX", command[5])
	require.Equal(t, entity.ExpiresAt.UnixMilli(), command[4])
}

func TestRedisRepositoryUpdateErrors(t *testing.T) {
	t.Parallel()

	backendErr := errors.New("redis unavailable")
	client := &redisClientStub{
		doFn: func(context.Context, ...any) *goredis.Cmd {
			return goredis.NewCmdResult(nil, goredis.Nil)
		},
	}

	repository := NewRedisRepository(client)
	require.EqualError(t, repository.Update(context.Background(), nil), "update authorization transaction: entity is nil")
	require.ErrorIs(t, repository.Update(context.Background(), transactionFixture()), domain.ErrNotFound)

	client.doFn = func(context.Context, ...any) *goredis.Cmd {
		return goredis.NewCmdResult(nil, backendErr)
	}
	require.ErrorIs(t, repository.Update(context.Background(), transactionFixture()), backendErr)
}

func TestRedisRepositoryDelete(t *testing.T) {
	t.Parallel()

	id := transactionFixture().ID
	backendErr := errors.New("redis unavailable")
	client := &redisClientStub{
		delFn: func(_ context.Context, keys ...string) *goredis.IntCmd {
			require.Equal(t, []string{key(id)}, keys)
			return goredis.NewIntResult(1, nil)
		},
	}
	repository := NewRedisRepository(client)

	require.NoError(t, repository.Delete(context.Background(), id))

	client.delFn = func(context.Context, ...string) *goredis.IntCmd {
		return goredis.NewIntResult(0, nil)
	}
	require.ErrorIs(t, repository.Delete(context.Background(), id), domain.ErrNotFound)

	client.delFn = func(context.Context, ...string) *goredis.IntCmd {
		return goredis.NewIntResult(0, backendErr)
	}
	require.ErrorIs(t, repository.Delete(context.Background(), id), backendErr)
}

func TestModelValidation(t *testing.T) {
	t.Parallel()

	entity := transactionFixture()
	value := toModel(entity)

	value.ID = "invalid"
	_, err := value.toDomain()
	require.ErrorContains(t, err, "parse identifier")

	value = toModel(entity)
	value.OAuthClientID = "invalid"
	_, err = value.toDomain()
	require.ErrorContains(t, err, "parse OAuth client identifier")

	value = toModel(entity)
	invalidIdentity := "invalid"
	value.IdentityID = &invalidIdentity
	_, err = value.toDomain()
	require.ErrorContains(t, err, "parse identity identifier")

	value = toModel(entity)
	value.ExpiresAt = time.Time{}
	_, err = value.toDomain()
	require.EqualError(t, err, "expiration is missing")
}

func TestEncodedModelUsesStringIdentifiers(t *testing.T) {
	t.Parallel()

	entity := transactionFixture()
	payload, err := encode(entity)
	require.NoError(t, err)

	var value map[string]any
	require.NoError(t, json.Unmarshal(payload, &value))
	require.Equal(t, entity.ID.UUID().String(), value["id"])
	require.Equal(t, entity.OAuthClientID.UUID().String(), value["oauth_client_id"])
	require.Equal(t, entity.IdentityID.UUID().String(), value["identity_id"])
}

func TestCodecPreservesVerificationReference(t *testing.T) {
	t.Parallel()

	entity := transactionFixture()
	entity.Status = domain.StatusPending
	entity.NextStep = domain.NextStepEmailVerification
	entity.AuthenticationLevel = nil
	entity.AuthorizationSource = nil
	entity.GrantedScopes = nil
	entity.Verification = &domain.VerificationReference{
		Channel:     domain.VerificationChannelEmail,
		ContactID:   uuid.MustParse("44444444-4444-4444-8444-444444444444"),
		ChallengeID: uuid.MustParse("55555555-5555-4555-8555-555555555555"),
		ExpiresAt:   entity.ExpiresAt.Add(-time.Minute),
	}

	payload, err := encode(entity)
	require.NoError(t, err)
	decoded, err := decode(payload)

	require.NoError(t, err)
	require.Equal(t, entity, decoded)
}

type redisClientStub struct {
	doFn  func(context.Context, ...any) *goredis.Cmd
	getFn func(context.Context, string) *goredis.StringCmd
	delFn func(context.Context, ...string) *goredis.IntCmd
}

func (s *redisClientStub) Do(ctx context.Context, args ...any) *goredis.Cmd {
	return s.doFn(ctx, args...)
}

func (s *redisClientStub) Get(ctx context.Context, key string) *goredis.StringCmd {
	return s.getFn(ctx, key)
}

func (s *redisClientStub) Del(ctx context.Context, keys ...string) *goredis.IntCmd {
	return s.delFn(ctx, keys...)
}

func transactionFixture() *domain.AuthorizationTransaction {
	authenticatedAt := time.Date(2026, time.August, 5, 12, 0, 0, 123000000, time.UTC)
	expiresAt := authenticatedAt.Add(10 * time.Minute)
	state := domain.State("oauth-state")
	nonce := openidconnectdomain.Nonce("oidc-nonce")
	pkce := oauthdomain.PKCE{
		Challenge: oauthdomain.CodeChallenge("challenge"),
		Method:    oauthdomain.CodeChallengeMethodS256,
	}
	identityID := identitydomain.ID(uuid.MustParse("22222222-2222-4222-8222-222222222222"))
	level := authenticationdomain.LevelMultiFactor
	source := domain.AuthorizationSourceUserConsent

	return &domain.AuthorizationTransaction{
		ID:            domain.ID(uuid.MustParse("11111111-1111-4111-8111-111111111111")),
		OAuthClientID: oauthclientdomain.ID(uuid.MustParse("33333333-3333-4333-8333-333333333333")),

		RedirectURI:     oauthclientdomain.RedirectURIValue("https://client.example/callback"),
		RequestedScopes: []oauthdomain.Scope{"openid", "profile"},
		State:           &state,
		Nonce:           &nonce,
		PKCE:            &pkce,

		Status:   domain.StatusReady,
		NextStep: domain.NextStepReady,

		IdentityID:           &identityID,
		AuthenticationLevel:  &level,
		AuthenticatedMethods: []authenticationdomain.Method{authenticationdomain.MethodPassword, authenticationdomain.MethodTOTP},
		GrantedScopes:        []oauthdomain.Scope{"openid"},
		AuthorizationSource:  &source,

		AuthenticatedAt: &authenticatedAt,
		ExpiresAt:       expiresAt,
	}
}
