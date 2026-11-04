package jwt

import (
	"crypto/ed25519"
	"path/filepath"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
	accesstokenservice "github.com/krackl1n/careeria/services/identity/internal/service/accesstoken"
)

const privateKeyRefForTest = "local://access-token-signing-key"

func TestSignerReturnsSignedAccessToken(t *testing.T) {
	t.Parallel()

	signer := newSignerForTest(t)

	issuedAt := time.Date(
		2026,
		time.August,
		7,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	input := accesstokenservice.SignInput{
		KeyID:         signingkeydomain.KeyID("identity-local-1"),
		Algorithm:     signingkeydomain.AlgorithmEdDSA,
		PrivateKeyRef: signingkeydomain.PrivateKeyRef(privateKeyRefForTest),
		Claims: accesstokenservice.Claims{
			Issuer:   "http://localhost:8080",
			Subject:  "identity-1",
			Audience: []string{"careeria-api"},

			ExpiresAt: issuedAt.Add(15 * time.Minute),
			IssuedAt:  issuedAt,
			TokenID:   "token-1",

			ClientID:  "web-client",
			Scopes:    []string{"openid", "profile"},
			SessionID: "session-1",

			AuthenticationLevel:  "multi_factor",
			AuthenticatedMethods: []string{"password", "totp"},
			AuthenticatedAt:      issuedAt.Add(-time.Minute),
		},
	}

	rawToken, err := signer.Sign(t.Context(), input)
	require.NoError(t, err)
	require.NotEmpty(t, rawToken)

	publicKey, ok := signer.privateKey.Public().(ed25519.PublicKey)
	require.True(t, ok)

	parsedClaims := &tokenClaims{}

	token, err := jwtlib.ParseWithClaims(
		rawToken,
		parsedClaims,
		func(_ *jwtlib.Token) (any, error) {
			return publicKey, nil
		},
		jwtlib.WithValidMethods([]string{
			jwtlib.SigningMethodEdDSA.Alg(),
		}),
		jwtlib.WithIssuer(input.Claims.Issuer),
		jwtlib.WithAudience(input.Claims.Audience[0]),
		jwtlib.WithExpirationRequired(),
		jwtlib.WithIssuedAt(),
		jwtlib.WithTimeFunc(func() time.Time {
			return issuedAt.Add(time.Minute)
		}),
	)
	require.NoError(t, err)
	require.True(t, token.Valid)

	require.Equal(t, accessTokenType, token.Header["typ"])
	require.Equal(t, jwtlib.SigningMethodEdDSA.Alg(), token.Header["alg"])
	require.Equal(t, string(input.KeyID), token.Header["kid"])

	require.Equal(t, input.Claims.Issuer, parsedClaims.Issuer)
	require.Equal(t, input.Claims.Subject, parsedClaims.Subject)
	require.Equal(
		t,
		jwtlib.ClaimStrings(input.Claims.Audience),
		parsedClaims.Audience,
	)

	require.NotNil(t, parsedClaims.ExpiresAt)
	require.True(t, input.Claims.ExpiresAt.Equal(parsedClaims.ExpiresAt.Time))

	require.NotNil(t, parsedClaims.IssuedAt)
	require.True(t, input.Claims.IssuedAt.Equal(parsedClaims.IssuedAt.Time))

	require.Equal(t, input.Claims.TokenID, parsedClaims.ID)
	require.Equal(t, input.Claims.ClientID, parsedClaims.ClientID)
	require.Equal(t, "openid profile", parsedClaims.Scopes)
	require.Equal(t, input.Claims.SessionID, parsedClaims.SessionID)

	require.Equal(
		t,
		input.Claims.AuthenticationLevel,
		parsedClaims.AuthenticationLevel,
	)
	require.Equal(
		t,
		input.Claims.AuthenticatedMethods,
		parsedClaims.AuthenticatedMethods,
	)

	require.NotNil(t, parsedClaims.AuthenticatedAt)
	require.True(t, input.Claims.AuthenticatedAt.Equal(parsedClaims.AuthenticatedAt.Time))
}

func TestSignerRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	signer := newSignerForTest(t)

	tests := []struct {
		name      string
		input     accesstokenservice.SignInput
		wantError string
	}{
		{
			name: "unknown private key reference",
			input: accesstokenservice.SignInput{
				KeyID:         signingkeydomain.KeyID("identity-local-1"),
				PrivateKeyRef: signingkeydomain.PrivateKeyRef("local://another-key"),
				Algorithm:     signingkeydomain.AlgorithmEdDSA,
			},
			wantError: "JWT signing key reference is not configured",
		},
		{
			name: "unsupported signing algorithm",
			input: accesstokenservice.SignInput{
				KeyID:         signingkeydomain.KeyID("identity-local-1"),
				PrivateKeyRef: signingkeydomain.PrivateKeyRef(privateKeyRefForTest),
				Algorithm:     signingkeydomain.AlgorithmES256,
			},
			wantError: "JWT signing algorithm is not supported",
		},
		{
			name: "missing signing key ID",
			input: accesstokenservice.SignInput{
				PrivateKeyRef: signingkeydomain.PrivateKeyRef(privateKeyRefForTest),
				Algorithm:     signingkeydomain.AlgorithmEdDSA,
			},
			wantError: "JWT signing key ID is required",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := signer.Sign(t.Context(), tc.input)

			require.EqualError(t, err, tc.wantError)
		})
	}
}

func newSignerForTest(t *testing.T) *Signer {
	t.Helper()

	signer, err := NewSigner(Config{
		KeyID:         "identity-local-1",
		PrivateKeyRef: privateKeyRefForTest,
		KeyFile: filepath.Join(
			"..",
			"..",
			"..",
			"configs",
			"access-token-signing-key.pem",
		),
	})
	require.NoError(t, err)

	return signer
}
