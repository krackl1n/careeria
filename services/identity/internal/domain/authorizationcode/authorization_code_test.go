package authorizationcode

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	openidconnectdomain "github.com/krackl1n/careeria/services/identity/internal/domain/openidconnect"
	"github.com/stretchr/testify/require"
)

func TestAuthorizationCodeCanBeValidatedForExchange(t *testing.T) {
	t.Parallel()

	now := authorizationCodeTestTime()
	code := newAuthorizationCodeForTest(t, now)
	verifier := oauthdomain.CodeVerifier(testCodeVerifier)
	require.NoError(t, code.ValidateForExchange(
		code.OAuthClientID,
		code.RedirectURI,
		&verifier,
		now.Add(time.Minute),
	))
}

func TestAuthorizationCodeIsBoundToOAuthClientAndRedirectURI(t *testing.T) {
	t.Parallel()

	now := authorizationCodeTestTime()
	code := newAuthorizationCodeForTest(t, now)
	verifier := oauthdomain.CodeVerifier(testCodeVerifier)

	err := code.ValidateForExchange(
		oauthclientdomain.ID(uuid.New()),
		code.RedirectURI,
		&verifier,
		now.Add(time.Minute),
	)
	require.ErrorIs(t, err, ErrOAuthClientMismatch)

	otherRedirectURI, err := oauthclientdomain.NewRedirectURI("https://client.example/other")
	require.NoError(t, err)
	err = code.ValidateForExchange(
		code.OAuthClientID,
		otherRedirectURI,
		&verifier,
		now.Add(time.Minute),
	)
	require.ErrorIs(t, err, ErrRedirectURIMismatch)
}

func TestAuthorizationCodeVerifiesPKCE(t *testing.T) {
	t.Parallel()

	now := authorizationCodeTestTime()
	code := newAuthorizationCodeForTest(t, now)
	err := code.ValidateForExchange(code.OAuthClientID, code.RedirectURI, nil, now.Add(time.Minute))
	require.ErrorIs(t, err, ErrPKCERequired)

	wrongVerifier, err := oauthdomain.NewCodeVerifier("another-verifier-that-is-valid-but-does-not-match-123456789")
	require.NoError(t, err)
	err = code.ValidateForExchange(code.OAuthClientID, code.RedirectURI, &wrongVerifier, now.Add(time.Minute))
	require.ErrorIs(t, err, ErrPKCEVerificationFailed)
}

func TestExpiredAuthorizationCodeCannotBeExchanged(t *testing.T) {
	t.Parallel()

	now := authorizationCodeTestTime()
	code := newAuthorizationCodeForTest(t, now)
	verifier := oauthdomain.CodeVerifier(testCodeVerifier)
	err := code.ValidateForExchange(code.OAuthClientID, code.RedirectURI, &verifier, code.ExpiresAt)
	require.ErrorIs(t, err, ErrExpired)
}

func newAuthorizationCodeForTest(t *testing.T, now time.Time) *AuthorizationCode {
	t.Helper()

	redirectURI, err := oauthclientdomain.NewRedirectURI("https://client.example/callback")
	require.NoError(t, err)
	challenge := codeChallengeFor(testCodeVerifier)
	method := "S256"
	pkce, err := oauthdomain.NewPKCE(&challenge, &method)
	require.NoError(t, err)
	nonce := openidconnectdomain.Nonce("oidc-nonce")
	code, err := New(
		Hash("authorization-code-hash"),
		oauthclientdomain.ID(uuid.New()),
		redirectURI,
		identitydomain.ID(uuid.New()),
		authenticationdomain.LevelMultiFactor,
		[]authenticationdomain.Method{authenticationdomain.MethodPassword, authenticationdomain.MethodTOTP},
		[]oauthdomain.Scope{"openid", "profile"},
		&nonce,
		pkce,
		now.Add(-time.Minute),
		now.Add(2*time.Minute),
		now,
	)
	require.NoError(t, err)
	return code
}

func codeChallengeFor(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func authorizationCodeTestTime() time.Time {
	return time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
}

const testCodeVerifier = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
