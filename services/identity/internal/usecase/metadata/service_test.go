package metadata

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
	"github.com/stretchr/testify/require"
)

func TestGetOpenIDConfiguration(t *testing.T) {
	t.Parallel()

	service := newServiceForTest(t, &signingKeyRepositoryStub{}, metadataTestTime())

	configuration := service.GetOpenIDConfiguration()

	require.Equal(t, OpenIDConfiguration{
		Issuer:                                 "https://identity.example.com",
		AuthorizationEndpoint:                  "https://identity.example.com/oauth2/authorize",
		TokenEndpoint:                          "https://identity.example.com/oauth2/token",
		RevocationEndpoint:                     "https://identity.example.com/oauth2/revoke",
		UserInfoEndpoint:                       "https://identity.example.com/oauth2/userinfo",
		JSONWebKeySetURI:                       "https://identity.example.com/.well-known/jwks.json",
		ResponseTypesSupported:                 []string{"code"},
		SubjectTypesSupported:                  []string{"public"},
		IDTokenSigningAlgorithmsSupported:      []string{"EdDSA"},
		ScopesSupported:                        []string{"openid", "profile", "email", "phone"},
		ClaimsSupported:                        []string{"sub", "name", "family_name", "given_name", "middle_name", "email", "email_verified", "phone_number", "phone_number_verified", "updated_at", "acr", "amr", "auth_time"},
		GrantTypesSupported:                    []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethodsSupported:      []string{"none", "client_secret_basic"},
		RevocationEndpointAuthMethodsSupported: []string{"none", "client_secret_basic"},
		CodeChallengeMethodsSupported:          []string{"S256"},
	}, configuration)
}

func TestGetJSONWebKeySetReturnsPublicVerificationKeys(t *testing.T) {
	t.Parallel()

	now := metadataTestTime()
	active := signingKeyForMetadataTest(now, signingkeydomain.StatusActive)
	retiring := signingKeyForMetadataTest(now, signingkeydomain.StatusRetiring)
	keys := &signingKeyRepositoryStub{keys: []*signingkeydomain.SigningKey{active, retiring}}
	service := newServiceForTest(t, keys, now)

	result, err := service.GetJSONWebKeySet(t.Context())
	require.NoError(t, err)
	require.Equal(t, []signingkeydomain.PublicJWK{active.PublicJWK, retiring.PublicJWK}, result.Keys)
	require.Equal(t, 1, keys.calls)
	require.Equal(t, now, keys.now)

	result.Keys[0][0] = '['
	require.Equal(t, byte('{'), active.PublicJWK[0])
}

func TestGetJSONWebKeySetRejectsMissingOrUnpublishableKeys(t *testing.T) {
	t.Parallel()

	now := metadataTestTime()
	tests := map[string][]*signingkeydomain.SigningKey{
		"missing": nil,
		"retired": {signingKeyForMetadataTest(now, signingkeydomain.StatusRetired)},
	}

	for name, keys := range tests {
		name, keys := name, keys
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			service := newServiceForTest(t, &signingKeyRepositoryStub{keys: keys}, now)
			_, err := service.GetJSONWebKeySet(t.Context())
			require.Error(t, err)
		})
	}
}

func TestGetJSONWebKeySetPreservesRepositoryError(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("database unavailable")
	service := newServiceForTest(
		t,
		&signingKeyRepositoryStub{err: repositoryErr},
		metadataTestTime(),
	)

	_, err := service.GetJSONWebKeySet(t.Context())
	require.ErrorIs(t, err, repositoryErr)
}

func TestNewServiceValidatesConfigurationAndDependencies(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Issuer:                  "https://identity.example.com",
		IDTokenSigningAlgorithm: "EdDSA",
	}
	keys := &signingKeyRepositoryStub{}
	clock := &clockStub{now: metadataTestTime()}

	_, err := NewService(Config{}, keys, clock)
	require.ErrorContains(t, err, "metadata config")
	_, err = NewService(cfg, nil, clock)
	require.ErrorContains(t, err, "signing key repository is required")
	_, err = NewService(cfg, keys, nil)
	require.ErrorContains(t, err, "clock is required")
}

type signingKeyRepositoryStub struct {
	keys  []*signingkeydomain.SigningKey
	err   error
	calls int
	now   time.Time
}

func (repository *signingKeyRepositoryStub) ListPublic(
	_ context.Context,
	now time.Time,
) ([]*signingkeydomain.SigningKey, error) {
	repository.calls++
	repository.now = now
	return repository.keys, repository.err
}

type clockStub struct{ now time.Time }

func (clock *clockStub) Now() time.Time { return clock.now }

func newServiceForTest(t *testing.T, keys SigningKeyRepository, now time.Time) *Service {
	t.Helper()

	service, err := NewService(
		Config{
			Issuer:                  "https://identity.example.com",
			IDTokenSigningAlgorithm: "EdDSA",
		},
		keys,
		&clockStub{now: now},
	)
	require.NoError(t, err)
	return service
}

func signingKeyForMetadataTest(
	now time.Time,
	status signingkeydomain.Status,
) *signingkeydomain.SigningKey {
	createdAt := now.Add(-time.Hour)
	activatedAt := now.Add(-30 * time.Minute)
	key := &signingkeydomain.SigningKey{
		ID:          signingkeydomain.ID(uuid.New()),
		KeyID:       signingkeydomain.KeyID(uuid.NewString()),
		Algorithm:   signingkeydomain.AlgorithmEdDSA,
		PublicJWK:   signingkeydomain.PublicJWK(`{"kty":"OKP","crv":"Ed25519","x":"public"}`),
		Status:      status,
		CreatedAt:   createdAt,
		NotBefore:   createdAt,
		ActivatedAt: &activatedAt,
	}
	if status == signingkeydomain.StatusRetiring {
		retiringAt := now.Add(-time.Minute)
		key.RetiringAt = &retiringAt
	}
	if status == signingkeydomain.StatusRetired {
		retiringAt := now.Add(-2 * time.Minute)
		retiredAt := now.Add(-time.Minute)
		key.RetiringAt = &retiringAt
		key.RetiredAt = &retiredAt
	}
	return key
}

func metadataTestTime() time.Time {
	return time.Date(2026, time.August, 8, 15, 0, 0, 0, time.UTC)
}
