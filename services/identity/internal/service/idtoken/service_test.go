package idtoken

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	openidconnectdomain "github.com/krackl1n/careeria/services/identity/internal/domain/openidconnect"
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
	"github.com/stretchr/testify/require"
)

func TestIssueReturnsSignedIDToken(t *testing.T) {
	now := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	keys := &keyRepositoryStub{key: &signingkeydomain.SigningKey{
		KeyID:         "identity-local-1",
		Algorithm:     signingkeydomain.AlgorithmEdDSA,
		PrivateKeyRef: "local://access-token-signing-key",
	}}
	signer := &signerStub{value: "id-token"}
	service, err := NewService(Config{
		Issuer:    "https://identity.example.com",
		Algorithm: "EdDSA",
		Lifetime:  15 * time.Minute,
	}, keys, signer, clockStub{now: now})
	require.NoError(t, err)
	nonce := openidconnectdomain.Nonce("nonce")
	identityID := identitydomain.ID(uuid.New())
	name := "Иван Иванов"
	email := "user@example.com"
	emailVerified := true

	token, err := service.Issue(context.Background(), IssueInput{
		IdentityID: identityID,
		ClientID:   "web-client",
		Nonce:      &nonce,
		Identity: IdentityClaims{
			Name:          &name,
			Email:         &email,
			EmailVerified: &emailVerified,
		},
		AuthenticationLevel:  authenticationdomain.LevelSingleFactor,
		AuthenticatedMethods: []authenticationdomain.Method{authenticationdomain.MethodPassword},
		AuthenticatedAt:      now.Add(-time.Minute),
	})

	require.NoError(t, err)
	require.Equal(t, "id-token", token.Value)
	require.Equal(t, now, token.IssuedAt)
	require.Equal(t, now.Add(15*time.Minute), token.ExpiresAt)
	require.Equal(t, identityID.UUID().String(), signer.input.Claims.Subject)
	require.Equal(t, "web-client", signer.input.Claims.Audience)
	require.Equal(t, "nonce", *signer.input.Claims.Nonce)
	require.Equal(t, name, *signer.input.Claims.Identity.Name)
	require.Equal(t, email, *signer.input.Claims.Identity.Email)
	require.True(t, *signer.input.Claims.Identity.EmailVerified)
	require.Equal(t, signingkeydomain.AlgorithmEdDSA, keys.algorithm)
}

type keyRepositoryStub struct {
	key       *signingkeydomain.SigningKey
	algorithm signingkeydomain.Algorithm
}

func (s *keyRepositoryStub) GetActive(_ context.Context, algorithm signingkeydomain.Algorithm, _ time.Time) (*signingkeydomain.SigningKey, error) {
	s.algorithm = algorithm
	return s.key, nil
}

type signerStub struct {
	value string
	input SignInput
}

func (s *signerStub) SignIDToken(_ context.Context, input SignInput) (string, error) {
	s.input = input
	return s.value, nil
}

type clockStub struct {
	now time.Time
}

func (s clockStub) Now() time.Time {
	return s.now
}
