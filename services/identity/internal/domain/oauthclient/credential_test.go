package oauthclient

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCredentialCreationRejectsInvalidExpiration(t *testing.T) {
	t.Parallel()

	now := testTime()
	_, err := newCredential(
		CredentialID(uuid.New()),
		mustSecretHash(t, "$argon2id$client-secret"),
		nil,
		nil,
		&now,
		now,
	)

	require.EqualError(t, err, "credential expiration must follow creation")
}

func TestNewConfidentialDelegatesCredentialValidation(t *testing.T) {
	t.Parallel()

	now := testTime()
	_, err := NewConfidential(
		ID(uuid.New()),
		mustClientID(t, uuid.NewString()),
		"Server App",
		nil,
		nil,
		RedirectURIID(uuid.New()),
		mustRedirectURI(t, "https://client.careeria.dev/oauth/callback"),
		CredentialID(uuid.New()),
		mustSecretHash(t, "$argon2id$client-secret"),
		nil,
		&now,
		now,
	)

	require.EqualError(t, err, "create OAuth client: credential expiration must follow creation")
}

func TestCredentialOwnsLifecycle(t *testing.T) {
	t.Parallel()

	now := testTime()
	expiresAt := now.Add(time.Hour)
	credential, err := newCredential(
		CredentialID(uuid.New()),
		mustSecretHash(t, "$argon2id$client-secret"),
		nil,
		nil,
		&expiresAt,
		now,
	)
	require.NoError(t, err)

	name := "  Primary credential  "
	changed, err := credential.Rename(&name, now.Add(time.Minute))
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, "Primary credential", *credential.Name)

	usedAt := now.Add(2 * time.Minute)
	require.NoError(t, credential.RecordUsed(usedAt))
	require.Equal(t, usedAt, *credential.LastUsedAt)
	require.True(t, credential.IsUsable(usedAt))

	revokedAt := now.Add(3 * time.Minute)
	require.NoError(t, credential.Revoke(revokedAt, "  rotation completed  "))
	require.Equal(t, revokedAt, *credential.RevokedAt)
	require.Equal(t, "rotation completed", *credential.RevokeReason)
	require.False(t, credential.IsUsable(revokedAt))
	require.ErrorIs(t, credential.Validate(revokedAt), ErrCredentialRevoked)
	require.ErrorIs(t, credential.Revoke(revokedAt, "another reason"), ErrCredentialRevoked)
}

func TestDeletingOAuthClientPreservesPreviouslyRevokedCredential(t *testing.T) {
	t.Parallel()

	now := testTime()
	client := newConfidentialClient(t, now, nil)
	previouslyRevokedID := client.Credentials[0].ID
	activeID := CredentialID(uuid.New())
	require.NoError(t, client.AddCredential(
		activeID,
		mustSecretHash(t, "$argon2id$active-client-secret"),
		nil,
		nil,
		nil,
		now.Add(time.Minute),
	))

	previouslyRevokedAt := now.Add(2 * time.Minute)
	require.NoError(t, client.RevokeCredential(previouslyRevokedID, "rotation completed", previouslyRevokedAt))
	deletedAt := now.Add(3 * time.Minute)
	require.NoError(t, client.Delete(deletedAt))

	require.Equal(t, previouslyRevokedAt, *client.Credentials[0].RevokedAt)
	require.Equal(t, "rotation completed", *client.Credentials[0].RevokeReason)
	require.Equal(t, deletedAt, *client.Credentials[1].RevokedAt)
	require.Equal(t, "oauth_client_deleted", *client.Credentials[1].RevokeReason)
}
