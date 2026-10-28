package openidconnect

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewNonce(t *testing.T) {
	t.Parallel()

	nonce, err := NewNonce(strings.Repeat("a", maxNonceLength))
	require.NoError(t, err)
	require.Len(t, nonce, maxNonceLength)

	_, err = NewNonce(strings.Repeat("a", maxNonceLength+1))
	require.EqualError(t, err, "OpenID Connect nonce exceeds 2048 bytes")
}
