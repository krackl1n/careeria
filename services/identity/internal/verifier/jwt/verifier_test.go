package jwt

import (
	"crypto/ed25519"
	"testing"

	jwtlib "github.com/golang-jwt/jwt/v5"
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
	accesstokenservice "github.com/krackl1n/careeria/services/identity/internal/service/accesstoken"
	"github.com/stretchr/testify/require"
)

func TestVerifierImplementsAccessTokenPort(t *testing.T) {
	t.Parallel()

	var verifier accesstokenservice.Verifier = Verifier{}
	require.NotNil(t, verifier)
}

func TestVerifierReturnsDomainKeyID(t *testing.T) {
	t.Parallel()

	value := tokenWithKeyIDForTest(t, "key-1")

	keyID, err := (Verifier{}).KeyID(value)
	require.NoError(t, err)
	require.Equal(t, signingkeydomain.KeyID("key-1"), keyID)
}

func TestVerifierRejectsInvalidDomainKeyID(t *testing.T) {
	t.Parallel()

	value := tokenWithKeyIDForTest(t, "invalid key")

	_, err := (Verifier{}).KeyID(value)
	require.Error(t, err)
}

func tokenWithKeyIDForTest(t *testing.T, keyID string) string {
	t.Helper()

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodEdDSA, jwtlib.MapClaims{})
	token.Header["kid"] = keyID
	value, err := token.SignedString(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)))
	require.NoError(t, err)
	return value
}
