package oauthclient

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	"github.com/stretchr/testify/require"
)

func TestNewPublicOAuthClient(t *testing.T) {
	t.Parallel()

	now := testTime()
	client, err := NewPublic(
		ID(uuid.New()),
		mustClientID(t, "mobile-app"),
		"Mobile App",
		nil,
		nil,
		RedirectURIID(uuid.New()),
		mustRedirectURI(t, "com.careeria.app:/oauth/callback"),
		now,
	)

	require.NoError(t, err)
	require.NoError(t, client.AddGrantType(GrantTypeRefreshToken, now))
	require.NoError(t, client.AddScope(mustScope(t, "openid"), now))
	require.NoError(t, client.AddScope(mustScope(t, "profile"), now))
	require.Equal(t, TokenEndpointAuthMethodNone, client.TokenEndpointAuthMethod)
	require.True(t, client.RequirePKCE)
	require.Equal(t, ConsentModeRemember, client.ConsentMode)
	require.Empty(t, client.Credentials)
	require.Equal(t, StatusActive, client.Status)
}

func TestOAuthClientConsentModeCanBeChanged(t *testing.T) {
	t.Parallel()

	now := testTime()
	client := newPublicClient(t, now)

	require.NoError(t, client.SetConsentMode(ConsentModeTrusted, now.Add(time.Minute)))
	require.Equal(t, ConsentModeTrusted, client.ConsentMode)
	require.Error(t, client.SetConsentMode(ConsentMode("unknown"), now.Add(2*time.Minute)))
	require.Equal(t, ConsentModeTrusted, client.ConsentMode)
}

func TestOAuthClientCredentialsFollowClientType(t *testing.T) {
	t.Parallel()

	now := testTime()
	client, err := NewConfidential(
		ID(uuid.New()),
		mustClientID(t, uuid.NewString()),
		"Server App",
		nil,
		nil,
		RedirectURIID(uuid.New()),
		mustRedirectURI(t, "https://client.careeria.dev/oauth/callback"),
		CredentialID(uuid.New()),
		mustSecretHash(t, "$argon2id$initial-client-secret"),
		nil,
		nil,
		now,
	)
	require.NoError(t, err)
	require.Equal(t, TokenEndpointAuthMethodClientSecretBasic, client.TokenEndpointAuthMethod)
	require.Len(t, client.Credentials, 1)

	publicClient := newPublicClient(t, now)
	err = publicClient.AddCredential(
		CredentialID(uuid.New()),
		mustSecretHash(t, "$argon2id$public-secret"),
		nil,
		nil,
		nil,
		now.Add(time.Minute),
	)
	require.ErrorIs(t, err, ErrCredentialsNotAllowed)
}

func TestPublicOAuthClientAlwaysRequiresPKCE(t *testing.T) {
	t.Parallel()

	now := testTime()
	client := newPublicClient(t, now)

	err := client.SetPKCERequired(false, now.Add(time.Minute))
	require.ErrorIs(t, err, ErrPKCERequired)
	require.True(t, client.RequirePKCE)
}

func TestOAuthClientRedirectURILifecycle(t *testing.T) {
	t.Parallel()

	now := testTime()
	client := newPublicClient(t, now)
	first := client.RedirectURIs[0]

	err := client.AddRedirectURI(RedirectURIID(uuid.New()), first.URI, now.Add(time.Minute))
	require.ErrorIs(t, err, ErrRedirectURIAlreadyExists)

	secondID := RedirectURIID(uuid.New())
	secondURI := mustRedirectURI(t, "https://app.careeria.dev/oauth/callback")
	require.NoError(t, client.AddRedirectURI(secondID, secondURI, now.Add(2*time.Minute)))
	require.NoError(t, client.RemoveRedirectURI(first.ID, now.Add(3*time.Minute)))
	require.Len(t, client.RedirectURIs, 1)
	require.Equal(t, secondURI, client.RedirectURIs[0].URI)

	err = client.RemoveRedirectURI(secondID, now.Add(4*time.Minute))
	require.ErrorIs(t, err, ErrLastRedirectURI)
}

func TestOAuthClientGrantAndScopeUniqueness(t *testing.T) {
	t.Parallel()

	now := testTime()
	client := newPublicClient(t, now)

	err := client.AddGrantType(GrantTypeAuthorizationCode, now.Add(time.Minute))
	require.ErrorIs(t, err, ErrGrantTypeAlreadyExists)
	err = client.RemoveGrantType(GrantTypeAuthorizationCode, now.Add(2*time.Minute))
	require.ErrorIs(t, err, ErrAuthorizationCodeRequired)

	openid := mustScope(t, "openid")
	require.NoError(t, client.AddScope(openid, now.Add(3*time.Minute)))
	err = client.AddScope(openid, now.Add(4*time.Minute))
	require.ErrorIs(t, err, ErrScopeAlreadyExists)
	require.NoError(t, client.RemoveScope(openid, now.Add(5*time.Minute)))
}

func TestConfidentialOAuthClientCredentialRotation(t *testing.T) {
	t.Parallel()

	now := testTime()
	client := newConfidentialClient(t, now, nil)
	firstID := client.Credentials[0].ID

	secondID := CredentialID(uuid.New())
	expiresAt := now.Add(24 * time.Hour)
	require.NoError(t, client.AddCredential(
		secondID,
		mustSecretHash(t, "$argon2id$rotated-client-secret"),
		nil,
		nil,
		&expiresAt,
		now.Add(time.Minute),
	))
	require.Len(t, client.Credentials, 2)

	require.NoError(t, client.RevokeCredential(firstID, "rotation completed", now.Add(2*time.Minute)))
	require.ErrorIs(t, client.ValidateCredential(firstID, now.Add(3*time.Minute)), ErrCredentialRevoked)
	require.NoError(t, client.ValidateCredential(secondID, now.Add(3*time.Minute)))
	require.NoError(t, client.RecordCredentialUsed(secondID, now.Add(4*time.Minute)))
	require.Equal(t, now.Add(4*time.Minute), *client.Credentials[1].LastUsedAt)
	require.ErrorIs(t, client.ValidateCredential(secondID, expiresAt), ErrCredentialExpired)
}

func TestDisabledConfidentialClientRequiresUsableCredentialToEnable(t *testing.T) {
	t.Parallel()

	now := testTime()
	expiresAt := now.Add(2 * time.Minute)
	client := newConfidentialClient(t, now, &expiresAt)
	require.NoError(t, client.Disable(now.Add(time.Minute)))

	err := client.Enable(expiresAt)
	require.ErrorIs(t, err, ErrUsableCredentialRequired)

	require.NoError(t, client.AddCredential(
		CredentialID(uuid.New()),
		mustSecretHash(t, "$argon2id$new-active-secret"),
		nil,
		nil,
		nil,
		now.Add(3*time.Minute),
	))
	require.NoError(t, client.Enable(now.Add(4*time.Minute)))
	require.Equal(t, StatusActive, client.Status)
	require.Nil(t, client.DisabledAt)
}

func TestDeletingOAuthClientRevokesCredentialsAndPreventsChanges(t *testing.T) {
	t.Parallel()

	now := testTime()
	client := newConfidentialClient(t, now, nil)
	deletedAt := now.Add(time.Minute)
	require.NoError(t, client.Delete(deletedAt))

	require.Equal(t, StatusDeleted, client.Status)
	require.Equal(t, deletedAt, *client.DeletedAt)
	require.Equal(t, deletedAt, *client.Credentials[0].RevokedAt)
	require.Equal(t, "oauth_client_deleted", *client.Credentials[0].RevokeReason)

	err := client.UpdateMetadata("Renamed", nil, now.Add(2*time.Minute))
	require.ErrorIs(t, err, ErrInvalidStatus)
}

func TestOAuthClientConstructorRejectsInvalidIdentifier(t *testing.T) {
	t.Parallel()

	now := testTime()
	_, err := NewPublic(
		ID(uuid.Nil),
		mustClientID(t, uuid.NewString()),
		"Invalid Client",
		nil,
		nil,
		RedirectURIID(uuid.New()),
		mustRedirectURI(t, "https://client.careeria.dev/oauth/callback"),
		now,
	)
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrNotFound))
}

func newPublicClient(t *testing.T, now time.Time) *OAuthClient {
	t.Helper()
	client, err := NewPublic(
		ID(uuid.New()),
		mustClientID(t, uuid.NewString()),
		"Test Client",
		nil,
		nil,
		RedirectURIID(uuid.New()),
		mustRedirectURI(t, "https://client.careeria.dev/oauth/callback"),
		now,
	)
	require.NoError(t, err)
	require.NoError(t, client.AddGrantType(GrantTypeRefreshToken, now))
	return client
}

func newConfidentialClient(t *testing.T, now time.Time, expiresAt *time.Time) *OAuthClient {
	t.Helper()
	client, err := NewConfidential(
		ID(uuid.New()),
		mustClientID(t, uuid.NewString()),
		"Test Client",
		nil,
		nil,
		RedirectURIID(uuid.New()),
		mustRedirectURI(t, "https://client.careeria.dev/oauth/callback"),
		CredentialID(uuid.New()),
		mustSecretHash(t, "$argon2id$initial-client-secret"),
		nil,
		expiresAt,
		now,
	)
	require.NoError(t, err)
	require.NoError(t, client.AddGrantType(GrantTypeRefreshToken, now))
	return client
}

func mustClientID(t *testing.T, value string) ClientID {
	t.Helper()
	result := ClientID(value)
	return result
}

func mustRedirectURI(t *testing.T, value string) RedirectURIValue {
	t.Helper()
	result, err := NewRedirectURI(value)
	require.NoError(t, err)
	return result
}

func mustScope(t *testing.T, value string) oauthdomain.Scope {
	t.Helper()
	result, err := oauthdomain.NewScope(value)
	require.NoError(t, err)
	return result
}

func mustSecretHash(t *testing.T, value string) SecretHash {
	t.Helper()
	result, err := NewSecretHash(value)
	require.NoError(t, err)
	return result
}

func testTime() time.Time {
	return time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
}
