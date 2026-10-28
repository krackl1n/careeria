package jwt

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestVerifyAccessToken(t *testing.T) {
	t.Parallel()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	now := jwtTestTime()
	claims := accessTokenClaimsForTest(now)
	value := signAccessTokenForTest(t, privateKey, "key-1", AccessTokenType, claims)

	keyID, err := ParseKeyID(value)
	require.NoError(t, err)
	require.Equal(t, "key-1", keyID)

	verified, err := VerifyAccessToken(VerifyAccessTokenInput{
		Value:     value,
		KeyID:     keyID,
		Algorithm: AlgorithmEdDSA,
		PublicJWK: publicJWKForTest(t, publicKey, keyID),
		Issuer:    claims.Issuer,
		Audience:  claims.Audience,
		Now:       now.Add(time.Minute),
	})
	require.NoError(t, err)
	require.Equal(t, claims, verified)
}

func TestVerifyAccessTokenRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	now := jwtTestTime()
	claims := accessTokenClaimsForTest(now)
	value := signAccessTokenForTest(t, privateKey, "key-1", AccessTokenType, claims)
	publicJWK := publicJWKForTest(t, publicKey, "key-1")

	parts := strings.Split(value, ".")
	require.Len(t, parts, 3)
	parts[1] = mutateBase64URL(parts[1])
	tampered := strings.Join(parts, ".")
	wrongType := signAccessTokenForTest(t, privateKey, "key-1", "JWT", claims)

	tests := map[string]VerifyAccessTokenInput{
		"signature":  {Value: tampered},
		"type":       {Value: wrongType},
		"issuer":     {Value: value, Issuer: "https://another.example.com"},
		"audience":   {Value: value, Audience: []string{"another-api"}},
		"expiration": {Value: value, Now: claims.ExpiresAt},
		"algorithm":  {Value: value, Algorithm: "RS256"},
	}

	for name, input := range tests {
		name, input := name, input
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			input.KeyID = "key-1"
			if input.Algorithm == "" {
				input.Algorithm = AlgorithmEdDSA
			}
			input.PublicJWK = publicJWK
			if input.Issuer == "" {
				input.Issuer = claims.Issuer
			}
			if input.Audience == nil {
				input.Audience = claims.Audience
			}
			if input.Now.IsZero() {
				input.Now = now.Add(time.Minute)
			}

			_, err := VerifyAccessToken(input)
			require.Error(t, err)
		})
	}
}

func TestParseKeyIDRejectsMalformedHeader(t *testing.T) {
	t.Parallel()

	_, err := ParseKeyID("not-a-jwt")
	require.Error(t, err)

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodEdDSA, jwtlib.MapClaims{})
	value, err := token.SignedString(privateKey)
	require.NoError(t, err)

	_, err = ParseKeyID(value)
	require.ErrorContains(t, err, "key ID is required")
}

func accessTokenClaimsForTest(now time.Time) AccessTokenClaims {
	return AccessTokenClaims{
		Issuer:   "https://identity.example.com",
		Subject:  "19c46302-4282-424a-8a10-2a032647158c",
		Audience: []string{"careeria-api", "careeria"},

		ExpiresAt: now.Add(15 * time.Minute),
		IssuedAt:  now,
		TokenID:   "169408bf-e044-4ab1-b070-b56aeb726eed",

		ClientID:  "web-client",
		Scopes:    []string{"openid", "profile"},
		SessionID: "66891c13-aa60-478e-994c-b1038a056e23",

		AuthenticationLevel:  "multi_factor",
		AuthenticatedMethods: []string{"password", "totp"},
		AuthenticatedAt:      now.Add(-time.Minute),
	}
}

func signAccessTokenForTest(
	t *testing.T,
	privateKey ed25519.PrivateKey,
	keyID string,
	tokenType string,
	claims AccessTokenClaims,
) string {
	t.Helper()

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodEdDSA, accessTokenClaims{
		RegisteredClaims: jwtlib.RegisteredClaims{
			Issuer:    claims.Issuer,
			Subject:   claims.Subject,
			Audience:  jwtlib.ClaimStrings(claims.Audience),
			ExpiresAt: jwtlib.NewNumericDate(claims.ExpiresAt),
			IssuedAt:  jwtlib.NewNumericDate(claims.IssuedAt),
			ID:        claims.TokenID,
		},
		ClientID:             claims.ClientID,
		Scopes:               strings.Join(claims.Scopes, " "),
		SessionID:            claims.SessionID,
		AuthenticationLevel:  claims.AuthenticationLevel,
		AuthenticatedMethods: claims.AuthenticatedMethods,
		AuthenticatedAt:      jwtlib.NewNumericDate(claims.AuthenticatedAt),
	})
	token.Header["typ"] = tokenType
	token.Header["kid"] = keyID
	value, err := token.SignedString(privateKey)
	require.NoError(t, err)
	return value
}

func publicJWKForTest(t *testing.T, publicKey ed25519.PublicKey, keyID string) []byte {
	t.Helper()

	value, err := json.Marshal(publicJWK{
		KeyType:   "OKP",
		Curve:     "Ed25519",
		X:         base64.RawURLEncoding.EncodeToString(publicKey),
		KeyID:     keyID,
		Algorithm: AlgorithmEdDSA,
		Use:       "sig",
	})
	require.NoError(t, err)
	return value
}

func mutateBase64URL(value string) string {
	if value[0] == 'A' {
		return "B" + value[1:]
	}
	return "A" + value[1:]
}

func jwtTestTime() time.Time {
	return time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
}
