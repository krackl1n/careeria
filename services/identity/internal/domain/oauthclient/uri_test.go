package oauthclient

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedirectURIRejectsBrowserExecutableSchemes(t *testing.T) {
	for _, value := range []string{
		"javascript:alert(document.domain)//",
		"data:text/html,<script>alert(1)</script>",
		"file:///etc/passwd",
	} {
		t.Run(value, func(t *testing.T) {
			_, err := NewRedirectURI(value)
			require.Error(t, err)
		})
	}
}

func TestRedirectURIAllowsSecureWebLoopbackAndPrivateUseSchemes(t *testing.T) {
	for _, value := range []string{
		"https://client.example/oauth/callback",
		"http://localhost:3000/oauth/callback",
		"http://127.0.0.1:4567/oauth/callback",
		"com.careeria.app:/oauth/callback",
	} {
		t.Run(value, func(t *testing.T) {
			redirectURI, err := NewRedirectURI(value)
			require.NoError(t, err)
			require.Equal(t, RedirectURIValue(value), redirectURI)
		})
	}
}

func TestRedirectURIRejectsInsecureNonLoopbackHTTP(t *testing.T) {
	_, err := NewRedirectURI("http://client.example/oauth/callback")
	require.Error(t, err)
}
