package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPKCEVerify(t *testing.T) {
	t.Parallel()

	verifier, err := NewCodeVerifier("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~")
	require.NoError(t, err)
	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])
	method := "S256"
	pkce, err := NewPKCE(&challenge, &method)
	require.NoError(t, err)

	require.True(t, pkce.Verify(verifier))
	require.False(t, pkce.Verify(CodeVerifier("different-verifier-value-that-is-long-enough-123456")))
}

func TestNewPKCE(t *testing.T) {
	t.Parallel()

	challenge := base64.RawURLEncoding.EncodeToString(make([]byte, sha256.Size))
	method := "S256"

	tests := []struct {
		name      string
		challenge *string
		method    *string
		wantNil   bool
	}{
		{name: "optional", wantNil: true},
		{name: "missing challenge", method: &method},
		{name: "missing method", challenge: &challenge},
		{name: "invalid challenge", challenge: stringPointer("invalid"), method: &method},
		{name: "unsupported method", challenge: &challenge, method: stringPointer("plain")},
		{name: "valid", challenge: &challenge, method: &method},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkce, err := NewPKCE(tt.challenge, tt.method)
			if tt.name == "valid" || tt.wantNil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			if tt.wantNil {
				require.Nil(t, pkce)
			}
		})
	}
}

func stringPointer(value string) *string {
	return &value
}
