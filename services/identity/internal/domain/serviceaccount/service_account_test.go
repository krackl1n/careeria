package serviceaccount

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNewServiceAccount(t *testing.T) {
	t.Parallel()

	now := serviceAccountTestTime()
	description := "  Meeting integrations  "
	credentialName := "  Primary secret  "
	prefix := mustSecretPrefix(t, "rvsa_live_7G2K")
	account, err := New(
		ID(uuid.New()),
		OrganizationID(uuid.New()),
		mustAccountKey(t, "meeting-bot"),
		"  Meeting Bot  ",
		&description,
		nil,
		CredentialID(uuid.New()),
		mustSecretHash(t, "$argon2id$initial-secret"),
		&prefix,
		&credentialName,
		nil,
		now,
	)

	require.NoError(t, err)
	require.Equal(t, "Meeting Bot", account.Name)
	require.Equal(t, "Meeting integrations", *account.Description)
	require.Equal(t, StatusActive, account.Status)
	require.Len(t, account.Credentials, 1)
	require.Equal(t, CredentialTypeSharedSecret, account.Credentials[0].Type)
	require.Equal(t, "Primary secret", *account.Credentials[0].Name)
	require.NoError(t, account.ValidateCredential(account.Credentials[0].ID, now))
}

func TestServiceAccountLifecycle(t *testing.T) {
	t.Parallel()

	now := serviceAccountTestTime()
	account := newServiceAccountForTest(t, now, nil)

	require.NoError(t, account.Disable(now.Add(time.Minute)))
	require.Equal(t, StatusDisabled, account.Status)
	require.ErrorIs(t, account.ValidateCredential(account.Credentials[0].ID, now.Add(2*time.Minute)), ErrInvalidStatus)

	require.NoError(t, account.Enable(now.Add(3*time.Minute)))
	require.Equal(t, StatusActive, account.Status)
	require.Nil(t, account.DisabledAt)

	description := "  Rotated integration account  "
	require.NoError(t, account.UpdateMetadata("  Integration Bot  ", &description, now.Add(4*time.Minute)))
	require.Equal(t, "Integration Bot", account.Name)
	require.Equal(t, "Rotated integration account", *account.Description)
}

func TestDeletedServiceAccountRevokesCredentials(t *testing.T) {
	t.Parallel()

	now := serviceAccountTestTime()
	account := newServiceAccountForTest(t, now, nil)
	secondID := CredentialID(uuid.New())
	require.NoError(t, account.AddCredential(
		secondID,
		mustSecretHash(t, "$argon2id$rotated-secret"),
		nil,
		nil,
		nil,
		nil,
		now.Add(time.Minute),
	))

	previouslyRevokedAt := now.Add(2 * time.Minute)
	require.NoError(t, account.RevokeCredential(account.Credentials[0].ID, "rotation completed", previouslyRevokedAt))

	deletedAt := now.Add(3 * time.Minute)
	require.NoError(t, account.Delete(deletedAt))
	require.Equal(t, StatusDeleted, account.Status)
	require.Equal(t, deletedAt, *account.DeletedAt)
	require.Equal(t, previouslyRevokedAt, *account.Credentials[0].RevokedAt)
	require.Equal(t, "rotation completed", *account.Credentials[0].RevokeReason)
	require.Equal(t, deletedAt, *account.Credentials[1].RevokedAt)
	require.Equal(t, "service_account_deleted", *account.Credentials[1].RevokeReason)

	err := account.UpdateMetadata("Renamed", nil, now.Add(4*time.Minute))
	require.ErrorIs(t, err, ErrInvalidStatus)
}

func TestDisabledServiceAccountRequiresUsableCredentialToEnable(t *testing.T) {
	t.Parallel()

	now := serviceAccountTestTime()
	expiresAt := now.Add(2 * time.Minute)
	account := newServiceAccountForTest(t, now, &expiresAt)
	require.NoError(t, account.Disable(now.Add(time.Minute)))

	err := account.Enable(expiresAt)
	require.ErrorIs(t, err, ErrUsableCredentialRequired)
	require.Equal(t, StatusDisabled, account.Status)
}

func TestNewServiceAccountRejectsInvalidIdentifiers(t *testing.T) {
	t.Parallel()

	now := serviceAccountTestTime()
	_, err := New(
		ID(uuid.Nil),
		OrganizationID(uuid.New()),
		mustAccountKey(t, "invalid-account"),
		"Invalid Account",
		nil,
		nil,
		CredentialID(uuid.New()),
		mustSecretHash(t, "$argon2id$secret"),
		nil,
		nil,
		nil,
		now,
	)
	require.Error(t, err)
}

func newServiceAccountForTest(t *testing.T, now time.Time, expiresAt *time.Time) *ServiceAccount {
	t.Helper()

	account, err := New(
		ID(uuid.New()),
		OrganizationID(uuid.New()),
		mustAccountKey(t, uuid.NewString()),
		"Integration Bot",
		nil,
		nil,
		CredentialID(uuid.New()),
		mustSecretHash(t, "$argon2id$initial-secret"),
		nil,
		nil,
		expiresAt,
		now,
	)
	require.NoError(t, err)

	return account
}

func mustAccountKey(t *testing.T, value string) AccountKey {
	t.Helper()

	result, err := NewAccountKey(value)
	require.NoError(t, err)

	return result
}

func mustSecretHash(t *testing.T, value string) SecretHash {
	t.Helper()

	result, err := NewSecretHash(value)
	require.NoError(t, err)

	return result
}

func mustSecretPrefix(t *testing.T, value string) SecretPrefix {
	t.Helper()

	result, err := NewSecretPrefix(value)
	require.NoError(t, err)

	return result
}

func serviceAccountTestTime() time.Time {
	return time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
}
