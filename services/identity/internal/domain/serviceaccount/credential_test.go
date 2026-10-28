package serviceaccount

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestServiceAccountCredentialRotation(t *testing.T) {
	t.Parallel()

	now := serviceAccountTestTime()
	account := newServiceAccountForTest(t, now, nil)
	initialID := account.Credentials[0].ID
	rotatedID := CredentialID(uuid.New())
	expiresAt := now.Add(24 * time.Hour)
	prefix := mustSecretPrefix(t, "rvsa_live_N4W8")
	name := "  Rotated secret  "

	require.NoError(t, account.AddCredential(
		rotatedID,
		mustSecretHash(t, "$argon2id$rotated-secret"),
		&prefix,
		&name,
		nil,
		&expiresAt,
		now.Add(time.Minute),
	))
	require.Len(t, account.Credentials, 2)
	require.Equal(t, "Rotated secret", *account.Credentials[1].Name)

	require.NoError(t, account.RevokeCredential(initialID, "rotation completed", now.Add(2*time.Minute)))
	require.ErrorIs(t, account.ValidateCredential(initialID, now.Add(3*time.Minute)), ErrCredentialRevoked)
	require.NoError(t, account.ValidateCredential(rotatedID, now.Add(3*time.Minute)))
	require.NoError(t, account.RecordCredentialUsed(rotatedID, now.Add(4*time.Minute)))
	require.Equal(t, now.Add(4*time.Minute), *account.Credentials[1].LastUsedAt)
	require.ErrorIs(t, account.ValidateCredential(rotatedID, expiresAt), ErrCredentialExpired)
}

func TestServiceAccountCredentialCannotBeUsedWhileAccountDisabled(t *testing.T) {
	t.Parallel()

	now := serviceAccountTestTime()
	account := newServiceAccountForTest(t, now, nil)
	credentialID := account.Credentials[0].ID
	require.NoError(t, account.Disable(now.Add(time.Minute)))

	err := account.RecordCredentialUsed(credentialID, now.Add(2*time.Minute))
	require.ErrorIs(t, err, ErrInvalidStatus)
	require.Nil(t, account.Credentials[0].LastUsedAt)
}

func TestAddCredentialRejectsInvalidExpiration(t *testing.T) {
	t.Parallel()

	now := serviceAccountTestTime()
	account := newServiceAccountForTest(t, now, nil)
	previousUpdatedAt := account.UpdatedAt
	previousCredentialCount := len(account.Credentials)

	err := account.AddCredential(
		CredentialID(uuid.New()),
		mustSecretHash(t, "$argon2id$invalid-credential"),
		nil,
		nil,
		nil,
		domainTimePointer(now),
		now.Add(time.Minute),
	)

	require.Error(t, err)
	require.Equal(t, previousUpdatedAt, account.UpdatedAt)
	require.Len(t, account.Credentials, previousCredentialCount)
}

func domainTimePointer(value time.Time) *time.Time {
	return &value
}
