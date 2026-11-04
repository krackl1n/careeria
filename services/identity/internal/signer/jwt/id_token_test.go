package jwt

import (
	"crypto/ed25519"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
	idtokenservice "github.com/krackl1n/careeria/services/identity/internal/service/idtoken"
	"github.com/stretchr/testify/require"
)

func TestSignerReturnsSignedIDToken(t *testing.T) {
	signer := newSignerForTest(t)
	issuedAt := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	nonce := "nonce"
	name := "Иван Иванов"
	email := "user@example.com"
	emailVerified := true
	updatedAt := issuedAt.Add(-2 * time.Minute)
	input := idtokenservice.SignInput{
		KeyID:         "identity-local-1",
		Algorithm:     signingkeydomain.AlgorithmEdDSA,
		PrivateKeyRef: privateKeyRefForTest,
		Claims: idtokenservice.Claims{
			Issuer:    "https://identity.example.com",
			Subject:   "identity-id",
			Audience:  "web-client",
			ExpiresAt: issuedAt.Add(15 * time.Minute),
			IssuedAt:  issuedAt,
			Nonce:     &nonce,
			Identity: idtokenservice.IdentityClaims{
				Name:          &name,
				Email:         &email,
				EmailVerified: &emailVerified,
				UpdatedAt:     &updatedAt,
			},
			AuthenticationLevel:  "single_factor",
			AuthenticatedMethods: []string{"password"},
			AuthenticatedAt:      issuedAt.Add(-time.Minute),
		},
	}

	raw, err := signer.SignIDToken(t.Context(), input)
	require.NoError(t, err)
	publicKey := signer.privateKey.Public().(ed25519.PublicKey)
	claims := &idTokenClaims{}
	token, err := jwtlib.ParseWithClaims(
		raw,
		claims,
		func(*jwtlib.Token) (any, error) { return publicKey, nil },
		jwtlib.WithValidMethods([]string{jwtlib.SigningMethodEdDSA.Alg()}),
		jwtlib.WithIssuer(input.Claims.Issuer),
		jwtlib.WithAudience(input.Claims.Audience),
		jwtlib.WithTimeFunc(func() time.Time { return issuedAt.Add(time.Minute) }),
	)

	require.NoError(t, err)
	require.True(t, token.Valid)
	require.Equal(t, idTokenType, token.Header["typ"])
	require.Equal(t, "identity-local-1", token.Header["kid"])
	require.Equal(t, input.Claims.Subject, claims.Subject)
	require.Equal(t, input.Claims.Nonce, claims.Nonce)
	require.Equal(t, input.Claims.Identity.Name, claims.Name)
	require.Equal(t, input.Claims.Identity.Email, claims.Email)
	require.Equal(t, input.Claims.Identity.EmailVerified, claims.EmailVerified)
	require.Equal(t, input.Claims.Identity.UpdatedAt.Unix(), claims.UpdatedAt.Unix())
	require.Equal(t, input.Claims.AuthenticationLevel, claims.AuthenticationLevel)
	require.Equal(t, input.Claims.AuthenticatedMethods, claims.AuthenticatedMethods)
}
