package passwordresetchallenge

import (
	"testing"
	"time"

	"github.com/google/uuid"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
	"github.com/stretchr/testify/require"
)

func TestPasswordResetChallengeRecordsFailedAttempts(t *testing.T) {
	t.Parallel()

	now := passwordResetChallengeTestTime()
	challenge := newPasswordResetChallengeForTest(t, 2, now)

	outcome, err := challenge.RecordFailedAttempt(now.Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, AttemptOutcomeRecorded, outcome)
	require.Equal(t, 1, challenge.Attempts)

	outcome, err = challenge.RecordFailedAttempt(now.Add(2 * time.Minute))
	require.NoError(t, err)
	require.Equal(t, AttemptOutcomeExhausted, outcome)
	require.Equal(t, 2, challenge.Attempts)

	_, err = challenge.RecordFailedAttempt(now.Add(3 * time.Minute))
	require.ErrorIs(t, err, ErrAttemptsExhausted)
	require.Equal(t, 2, challenge.Attempts)
}

func TestPasswordResetChallengeCanBeConsumedOnce(t *testing.T) {
	t.Parallel()

	now := passwordResetChallengeTestTime()
	challenge := newPasswordResetChallengeForTest(t, 3, now)
	consumedFromIP, err := sessiondomain.NewIPAddress("198.51.100.20")
	require.NoError(t, err)
	consumedAt := now.Add(time.Minute)

	require.NoError(t, challenge.Consume(&consumedFromIP, consumedAt))
	require.Equal(t, 1, challenge.Attempts)
	require.Equal(t, consumedAt, *challenge.LastAttemptAt)
	require.Equal(t, consumedAt, *challenge.ConsumedAt)
	require.Equal(t, consumedFromIP, *challenge.ConsumedFromIP)

	err = challenge.Consume(&consumedFromIP, now.Add(2*time.Minute))
	require.ErrorIs(t, err, ErrConsumed)
	require.Equal(t, 1, challenge.Attempts)
}

func TestExpiredPasswordResetChallengeCannotBeAttempted(t *testing.T) {
	t.Parallel()

	now := passwordResetChallengeTestTime()
	challenge := newPasswordResetChallengeForTest(t, 3, now)

	_, err := challenge.RecordFailedAttempt(challenge.ExpiresAt)
	require.ErrorIs(t, err, ErrExpired)
	require.Zero(t, challenge.Attempts)
}

func TestPasswordResetChallengeCanBeRevoked(t *testing.T) {
	t.Parallel()

	now := passwordResetChallengeTestTime()
	challenge := newPasswordResetChallengeForTest(t, 3, now)
	revokedAt := now.Add(time.Minute)

	require.NoError(t, challenge.Revoke(revokedAt))
	require.Equal(t, revokedAt, *challenge.RevokedAt)

	_, err := challenge.RecordFailedAttempt(now.Add(2 * time.Minute))
	require.ErrorIs(t, err, ErrRevoked)
	require.NoError(t, challenge.Revoke(now.Add(3*time.Minute)))
	require.Equal(t, revokedAt, *challenge.RevokedAt)
}

func TestConsumedPasswordResetChallengeCannotBeRevoked(t *testing.T) {
	t.Parallel()

	now := passwordResetChallengeTestTime()
	challenge := newPasswordResetChallengeForTest(t, 3, now)
	require.NoError(t, challenge.Consume(nil, now.Add(time.Minute)))

	err := challenge.Revoke(now.Add(2 * time.Minute))
	require.ErrorIs(t, err, ErrConsumed)
	require.Nil(t, challenge.RevokedAt)
}

func newPasswordResetChallengeForTest(t *testing.T, maxAttempts int, now time.Time) *PasswordResetChallenge {
	t.Helper()

	requestedFromIP, err := sessiondomain.NewIPAddress("192.0.2.10")
	require.NoError(t, err)
	challenge, err := New(
		ID(uuid.New()),
		identitydomain.ID(uuid.New()),
		TokenHash("reset-token-hash"),
		maxAttempts,
		now.Add(10*time.Minute),
		&requestedFromIP,
		now,
	)
	require.NoError(t, err)

	return challenge
}

func passwordResetChallengeTestTime() time.Time {
	return time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
}
