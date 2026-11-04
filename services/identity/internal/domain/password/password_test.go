package password

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	"github.com/stretchr/testify/require"
)

func TestPasswordFailedAttemptsAndLockLifecycle(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	entity := newPassword(t, now, nil)
	policy := FailurePolicy{MaxAttempts: 2, LockDuration: 15 * time.Minute}

	locked, err := entity.RecordFailedAttempt(policy, now.Add(time.Minute))
	require.NoError(t, err)
	require.False(t, locked)
	require.Equal(t, 1, entity.AuthState.FailedAttempts)

	locked, err = entity.RecordFailedAttempt(policy, now.Add(2*time.Minute))
	require.NoError(t, err)
	require.True(t, locked)
	require.Equal(t, 2, entity.AuthState.FailedAttempts)
	require.NotNil(t, entity.AuthState.LockedUntil)
	require.NotNil(t, entity.AuthState.LockReason)
	require.Equal(t, LockReasonFailedAttempts, *entity.AuthState.LockReason)

	err = entity.ensureAuthenticationAllowed(now.Add(3 * time.Minute))
	require.ErrorIs(t, err, ErrLocked)
	_, err = entity.RecordFailedAttempt(policy, now.Add(3*time.Minute))
	require.ErrorIs(t, err, ErrLocked)

	locked, err = entity.RecordFailedAttempt(policy, *entity.AuthState.LockedUntil)
	require.NoError(t, err)
	require.False(t, locked)
	require.Equal(t, 1, entity.AuthState.FailedAttempts)
	require.Nil(t, entity.AuthState.LockedUntil)
	require.Nil(t, entity.AuthState.LockReason)
}

func TestSuccessfulAuthenticationResetsFailedAttempts(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	entity := newPassword(t, now, nil)
	_, err := entity.RecordFailedAttempt(
		FailurePolicy{MaxAttempts: 3, LockDuration: 15 * time.Minute},
		now.Add(time.Minute),
	)
	require.NoError(t, err)

	require.NoError(t, entity.RecordSuccessfulAuthentication(now.Add(2*time.Minute)))
	require.Zero(t, entity.AuthState.FailedAttempts)
	require.Nil(t, entity.AuthState.LastFailedAt)
	require.Nil(t, entity.AuthState.LockedUntil)
}

func TestChangePasswordResetsCompromiseAndAuthenticationState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	entity := newPassword(t, now, nil)
	_, err := entity.RecordFailedAttempt(
		FailurePolicy{MaxAttempts: 3, LockDuration: 15 * time.Minute},
		now.Add(time.Minute),
	)
	require.NoError(t, err)
	require.NoError(t, entity.MarkCompromised(now.Add(2*time.Minute)))
	require.ErrorIs(t, entity.ensureAuthenticationAllowed(now.Add(3*time.Minute)), ErrCompromised)

	expiresAt := now.Add(90 * 24 * time.Hour)
	newHash := mustHash(t, "$argon2id$new-hash")
	require.NoError(t, entity.Change(newHash, false, &expiresAt, now.Add(4*time.Minute)))

	require.Equal(t, newHash, entity.Credential.Hash)
	require.False(t, entity.Credential.MustChange)
	require.Nil(t, entity.Credential.CompromisedAt)
	require.Equal(t, now.Add(4*time.Minute), entity.Credential.PasswordUpdatedAt)
	require.Zero(t, entity.AuthState.FailedAttempts)
	require.Nil(t, entity.AuthState.LastFailedAt)
	require.NoError(t, entity.ensureAuthenticationAllowed(now.Add(5*time.Minute)))
}

func TestPasswordExpiration(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)
	entity := newPassword(t, now, &expiresAt)

	require.NoError(t, entity.ensureAuthenticationAllowed(expiresAt.Add(-time.Nanosecond)))
	require.ErrorIs(t, entity.ensureAuthenticationAllowed(expiresAt), ErrExpired)
}

func TestRequirePasswordChangeIsIdempotent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	entity := newPassword(t, now, nil)

	require.NoError(t, entity.RequireChange(now.Add(time.Minute)))
	updatedAt := entity.Credential.UpdatedAt
	require.True(t, entity.Credential.MustChange)
	require.NoError(t, entity.RequireChange(now.Add(2*time.Minute)))
	require.Equal(t, updatedAt, entity.Credential.UpdatedAt)
}

func TestPasswordRejectsInvalidFailurePolicy(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	entity := newPassword(t, now, nil)

	_, err := entity.RecordFailedAttempt(FailurePolicy{}, now.Add(time.Minute))
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrLocked))
	require.Zero(t, entity.AuthState.FailedAttempts)
}

func newPassword(t *testing.T, now time.Time, expiresAt *time.Time) *Password {
	t.Helper()
	entity, err := New(
		identitydomain.ID(uuid.New()),
		mustHash(t, "$argon2id$initial-hash"),
		false,
		expiresAt,
		now,
	)
	require.NoError(t, err)
	return entity
}

func mustHash(t *testing.T, value string) Hash {
	t.Helper()
	hash, err := NewHash(value)
	require.NoError(t, err)
	return hash
}
