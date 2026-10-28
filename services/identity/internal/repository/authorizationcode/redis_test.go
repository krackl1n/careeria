package authorizationcode

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationcode"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	openidconnectdomain "github.com/krackl1n/careeria/services/identity/internal/domain/openidconnect"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRedisRepositoryCreate(t *testing.T) {
	t.Parallel()

	entity := authorizationCodeFixture()
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
	require.Equal(t, key(entity.Hash), command[1])
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

	entity := authorizationCodeFixture()
	backendErr := errors.New("redis unavailable")

	tests := []struct {
		name    string
		entity  *domain.AuthorizationCode
		result  *goredis.Cmd
		wantErr error
	}{
		{
			name:    "nil entity",
			wantErr: errors.New("create authorization code: entity is nil"),
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

func TestRedisRepositoryConsume(t *testing.T) {
	t.Parallel()

	entity := authorizationCodeFixture()
	payload, err := encode(entity)
	require.NoError(t, err)
	var calls int
	client := &redisClientStub{
		getDelFn: func(_ context.Context, gotKey string) *goredis.StringCmd {
			calls++
			require.Equal(t, key(entity.Hash), gotKey)
			return goredis.NewStringResult(string(payload), nil)
		},
	}

	got, err := NewRedisRepository(client).Consume(context.Background(), entity.Hash)

	require.NoError(t, err)
	require.Equal(t, entity, got)
	require.Equal(t, 1, calls)
}

func TestRedisRepositoryConsumeErrors(t *testing.T) {
	t.Parallel()

	hash := authorizationCodeFixture().Hash
	backendErr := errors.New("redis unavailable")
	mismatched := authorizationCodeFixture()
	mismatched.Hash = domain.Hash("another-hash")
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
			wantErr: errors.New("decode authorization code"),
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
			name:    "hash does not match key",
			result:  goredis.NewStringResult(string(mismatchedPayload), nil),
			wantErr: errors.New("stored hash does not match key"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &redisClientStub{
				getDelFn: func(context.Context, string) *goredis.StringCmd {
					return tt.result
				},
			}

			_, err := NewRedisRepository(client).Consume(context.Background(), hash)

			if tt.name == "invalid JSON" ||
				tt.name == "unsupported model" ||
				tt.name == "hash does not match key" {
				require.ErrorContains(t, err, tt.wantErr.Error())
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestModelValidation(t *testing.T) {
	t.Parallel()

	entity := authorizationCodeFixture()
	value := toModel(entity)

	value.OAuthClientID = "invalid"
	_, err := value.toDomain()
	require.ErrorContains(t, err, "parse OAuth client identifier")

	value = toModel(entity)
	value.IdentityID = "invalid"
	_, err = value.toDomain()
	require.ErrorContains(t, err, "parse identity identifier")

	value = toModel(entity)
	value.ExpiresAt = time.Time{}
	_, err = value.toDomain()
	require.EqualError(t, err, "expiration is missing")
}

func TestEncodedModelUsesStringIdentifiers(t *testing.T) {
	t.Parallel()

	entity := authorizationCodeFixture()
	payload, err := encode(entity)
	require.NoError(t, err)

	var value map[string]any
	require.NoError(t, json.Unmarshal(payload, &value))
	require.Equal(t, entity.OAuthClientID.UUID().String(), value["oauth_client_id"])
	require.Equal(t, entity.IdentityID.UUID().String(), value["identity_id"])
}

type redisClientStub struct {
	doFn     func(context.Context, ...any) *goredis.Cmd
	getDelFn func(context.Context, string) *goredis.StringCmd
}

func (s *redisClientStub) Do(ctx context.Context, args ...any) *goredis.Cmd {
	return s.doFn(ctx, args...)
}

func (s *redisClientStub) GetDel(ctx context.Context, key string) *goredis.StringCmd {
	return s.getDelFn(ctx, key)
}

func authorizationCodeFixture() *domain.AuthorizationCode {
	authenticatedAt := time.Date(2026, time.August, 5, 12, 0, 0, 123000000, time.UTC)
	issuedAt := authenticatedAt.Add(time.Minute)
	nonce := openidconnectdomain.Nonce("oidc-nonce")
	pkce := oauthdomain.PKCE{
		Challenge: oauthdomain.CodeChallenge("challenge"),
		Method:    oauthdomain.CodeChallengeMethodS256,
	}

	return &domain.AuthorizationCode{
		Hash:          domain.Hash("authorization-code-hash"),
		OAuthClientID: oauthclientdomain.ID(uuid.MustParse("33333333-3333-4333-8333-333333333333")),
		IdentityID:    identitydomain.ID(uuid.MustParse("22222222-2222-4222-8222-222222222222")),

		RedirectURI: oauthclientdomain.RedirectURIValue("https://client.example/callback"),
		Scopes:      []oauthdomain.Scope{"openid", "profile"},
		Nonce:       &nonce,
		PKCE:        &pkce,

		AuthenticationLevel:  authenticationdomain.LevelMultiFactor,
		AuthenticatedMethods: []authenticationdomain.Method{authenticationdomain.MethodPassword, authenticationdomain.MethodTOTP},

		AuthenticatedAt: authenticatedAt,
		IssuedAt:        issuedAt,
		ExpiresAt:       issuedAt.Add(5 * time.Minute),
	}
}
