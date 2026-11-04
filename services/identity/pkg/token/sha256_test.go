package token

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSHA256_Digest(t *testing.T) {
	digester := NewSHA256()

	value := "some-random-token"

	sum := sha256.Sum256([]byte(value))
	expected := base64.RawURLEncoding.EncodeToString(sum[:])

	actual := digester.Digest(value)

	assert.Equal(t, expected, actual)
}

func TestSHA256_DigestDeterministic(t *testing.T) {
	digester := NewSHA256()

	first := digester.Digest("some-random-token")
	second := digester.Digest("some-random-token")

	assert.Equal(t, first, second)
}

func TestSHA256_DigestDifferentValues(t *testing.T) {
	digester := NewSHA256()

	first := digester.Digest("token-one")
	second := digester.Digest("token-two")

	assert.NotEqual(t, first, second)
}

func TestSHA256_DigestLength(t *testing.T) {
	digester := NewSHA256()

	digest := digester.Digest("some-random-token")

	assert.Len(t, digest, base64.RawURLEncoding.EncodedLen(sha256.Size))
}

func TestSHA256_DigestEmptyToken(t *testing.T) {
	digester := NewSHA256()

	digest := digester.Digest("")

	assert.NotEmpty(t, digest)
	assert.Len(t, digest, base64.RawURLEncoding.EncodedLen(sha256.Size))
}
