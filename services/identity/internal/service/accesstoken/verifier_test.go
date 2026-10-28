package accesstoken

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
	"github.com/stretchr/testify/require"
)

func TestVerifyReturnsDomainToken(t *testing.T) {
	t.Parallel()

	now := accessTokenTestTime()
	key := activeSigningKeyForTest(now)
	tokenID := uuid.New()
	identityID := uuid.New()
	sessionID := uuid.New()
	keys := &signingKeyRepositoryStub{key: key}
	verifier := &verifierStub{
		keyID: key.KeyID,
		claims: Claims{
			TokenID:   tokenID.String(),
			Subject:   identityID.String(),
			SessionID: sessionID.String(),
			ClientID:  "web-client",
			Scopes:    []string{"openid", "profile"},
		},
	}
	service := newVerificationServiceForTest(t, keys, verifier, now)

	token, err := service.Verify(t.Context(), signedTokenForTest)
	require.NoError(t, err)
	require.Equal(t, tokenID, token.ID)
	require.Equal(t, identityID, token.IdentityID.UUID())
	require.Equal(t, sessionID, token.SessionID.UUID())
	require.Equal(t, oauthclientdomain.ClientID("web-client"), token.ClientID)
	require.Equal(t, []oauthdomain.Scope{"openid", "profile"}, token.Scopes)
	require.Equal(t, 1, verifier.keyIDCalls)
	require.Equal(t, signedTokenForTest, verifier.value)
	require.Equal(t, 1, keys.getCalls)
	require.Equal(t, key.KeyID, keys.keyID)
	require.Equal(t, 1, verifier.verifyCalls)
	require.Equal(t, VerifyInput{
		Value:     signedTokenForTest,
		KeyID:     key.KeyID,
		Algorithm: key.Algorithm,
		PublicJWK: key.PublicJWK,
		Issuer:    "https://identity.example.com",
		Audience:  []string{"https://api.example.com", "careeria"},
		Now:       now,
	}, verifier.input)
}

func TestVerifyReturnsInvalidTokenForUntrustedData(t *testing.T) {
	t.Parallel()

	now := accessTokenTestTime()
	tests := map[string]struct {
		keys     *signingKeyRepositoryStub
		verifier *verifierStub
	}{
		"malformed JWT": {
			keys:     &signingKeyRepositoryStub{},
			verifier: &verifierStub{keyIDErr: errors.New("malformed JWT")},
		},
		"unknown key": {
			keys:     &signingKeyRepositoryStub{err: signingkeydomain.ErrNotFound},
			verifier: &verifierStub{keyID: signingkeydomain.KeyID("unknown-key")},
		},
		"retired key": {
			keys: &signingKeyRepositoryStub{key: retiredSigningKeyForTest(now)},
			verifier: &verifierStub{
				keyID: signingkeydomain.KeyID("identity-2026-08"),
			},
		},
		"invalid signature": {
			keys: &signingKeyRepositoryStub{key: activeSigningKeyForTest(now)},
			verifier: &verifierStub{
				keyID:     signingkeydomain.KeyID("identity-2026-08"),
				verifyErr: errors.New("signature is invalid"),
			},
		},
	}

	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			service := newVerificationServiceForTest(t, test.keys, test.verifier, now)
			_, err := service.Verify(t.Context(), "access-token")
			require.ErrorIs(t, err, ErrInvalidToken)
		})
	}
}

func TestVerifyPreservesSigningKeyRepositoryFailure(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("database unavailable")
	keys := &signingKeyRepositoryStub{err: repositoryErr}
	verifier := &verifierStub{keyID: signingkeydomain.KeyID("identity-2026-08")}
	service := newVerificationServiceForTest(t, keys, verifier, accessTokenTestTime())

	_, err := service.Verify(t.Context(), "access-token")
	require.ErrorIs(t, err, repositoryErr)
	require.NotErrorIs(t, err, ErrInvalidToken)
}

func retiredSigningKeyForTest(now time.Time) *signingkeydomain.SigningKey {
	key := activeSigningKeyForTest(now)
	key.Status = signingkeydomain.StatusRetired
	retiredAt := now.Add(-time.Minute)
	key.RetiredAt = &retiredAt
	return key
}

func newVerificationServiceForTest(
	t *testing.T,
	keys SigningKeyRepository,
	verifier Verifier,
	now time.Time,
) *Service {
	t.Helper()

	service, err := NewService(
		Config{
			Issuer:    "https://identity.example.com",
			Audience:  []string{"https://api.example.com", "careeria"},
			Algorithm: string(signingkeydomain.AlgorithmEdDSA),
			Lifetime:  15 * time.Minute,
		},
		keys,
		&signerStub{},
		verifier,
		&idGeneratorStub{value: uuid.New()},
		&clockStub{value: now},
	)
	require.NoError(t, err)
	return service
}
