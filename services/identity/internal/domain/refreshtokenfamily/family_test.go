package refreshtokenfamily

import (
	"testing"
	"time"

	"github.com/google/uuid"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
	"github.com/stretchr/testify/require"
)

func TestNewRefreshTokenFamily(t *testing.T) {
	t.Parallel()

	now := refreshTokenTestTime()
	ipAddress, err := sessiondomain.NewIPAddress("2001:0db8::1")
	require.NoError(t, err)
	userAgent := "  browser/1.0  "
	initialTokenID := TokenID(uuid.New())

	entity, err := New(
		ID(uuid.New()),
		sessiondomain.ID(uuid.New()),
		initialTokenID,
		TokenHash("initial-hash"),
		now.Add(time.Hour),
		now.Add(24*time.Hour),
		&ipAddress,
		&userAgent,
		now,
	)
	require.NoError(t, err)
	require.Len(t, entity.Tokens, 1)
	require.Equal(t, initialTokenID, entity.Tokens[0].ID)
	require.Nil(t, entity.Tokens[0].ParentTokenID)
	require.Equal(t, sessiondomain.IPAddress("2001:db8::1"), *entity.Tokens[0].CreatedFromIP)
	require.Equal(t, "browser/1.0", *entity.Tokens[0].CreatedUserAgent)
	require.Nil(t, entity.LastRotatedAt)
	require.Nil(t, entity.RevokedAt)
}

func TestRefreshTokenRotationBuildsParentChain(t *testing.T) {
	t.Parallel()

	now := refreshTokenTestTime()
	initialTokenID := TokenID(uuid.New())
	entity := newRefreshTokenFamilyForTest(t, initialTokenID, now, now.Add(time.Hour), now.Add(24*time.Hour))
	nextTokenID := TokenID(uuid.New())
	rotatedAt := now.Add(5 * time.Minute)

	outcome, err := entity.Rotate(
		initialTokenID,
		nextTokenID,
		TokenHash("next-hash"),
		now.Add(2*time.Hour),
		nil,
		nil,
		rotatedAt,
	)
	require.NoError(t, err)
	require.Equal(t, RotationOutcomeRotated, outcome)
	require.Len(t, entity.Tokens, 2)
	require.Equal(t, rotatedAt, *entity.Tokens[0].RotatedAt)
	require.Equal(t, initialTokenID, *entity.Tokens[1].ParentTokenID)
	require.Equal(t, rotatedAt, entity.Tokens[1].IssuedAt)
	require.Equal(t, rotatedAt, *entity.LastRotatedAt)
}

func TestRefreshTokenReuseRevokesFamilyAndReturnsOutcome(t *testing.T) {
	t.Parallel()

	now := refreshTokenTestTime()
	initialTokenID := TokenID(uuid.New())
	entity := newRefreshTokenFamilyForTest(t, initialTokenID, now, now.Add(time.Hour), now.Add(24*time.Hour))
	_, err := entity.Rotate(
		initialTokenID,
		TokenID(uuid.New()),
		TokenHash("next-hash"),
		now.Add(2*time.Hour),
		nil,
		nil,
		now.Add(5*time.Minute),
	)
	require.NoError(t, err)

	detectedAt := now.Add(10 * time.Minute)
	outcome, err := entity.Rotate(
		initialTokenID,
		TokenID(uuid.New()),
		TokenHash("must-not-be-created"),
		now.Add(3*time.Hour),
		nil,
		nil,
		detectedAt,
	)
	require.NoError(t, err)
	require.Equal(t, RotationOutcomeReuseDetected, outcome)
	require.Len(t, entity.Tokens, 2)
	require.Equal(t, detectedAt, *entity.ReuseDetectedAt)
	require.Equal(t, detectedAt, *entity.RevokedAt)
	require.Equal(t, reuseDetectedRevokeReason, *entity.RevokeReason)
	require.Equal(t, detectedAt, *entity.Tokens[0].ReuseDetectedAt)
	for _, token := range entity.Tokens {
		require.Equal(t, detectedAt, *token.RevokedAt)
		require.Equal(t, reuseDetectedRevokeReason, *token.RevokeReason)
	}

	_, err = entity.Rotate(
		entity.Tokens[1].ID,
		TokenID(uuid.New()),
		TokenHash("another-hash"),
		now.Add(4*time.Hour),
		nil,
		nil,
		now.Add(11*time.Minute),
	)
	require.ErrorIs(t, err, ErrRevoked)
}

func TestRefreshTokenRotationRejectsExpiredToken(t *testing.T) {
	t.Parallel()

	now := refreshTokenTestTime()
	initialTokenID := TokenID(uuid.New())
	expiresAt := now.Add(10 * time.Minute)
	entity := newRefreshTokenFamilyForTest(t, initialTokenID, now, expiresAt, now.Add(24*time.Hour))

	_, err := entity.Rotate(
		initialTokenID,
		TokenID(uuid.New()),
		TokenHash("next-hash"),
		now.Add(time.Hour),
		nil,
		nil,
		expiresAt,
	)
	require.ErrorIs(t, err, ErrTokenExpired)
	require.Len(t, entity.Tokens, 1)
	require.Nil(t, entity.Tokens[0].RotatedAt)
}

func TestInvalidRotationDoesNotMutateFamily(t *testing.T) {
	t.Parallel()

	now := refreshTokenTestTime()
	initialTokenID := TokenID(uuid.New())
	absoluteExpiresAt := now.Add(24 * time.Hour)
	entity := newRefreshTokenFamilyForTest(t, initialTokenID, now, now.Add(time.Hour), absoluteExpiresAt)

	_, err := entity.Rotate(
		initialTokenID,
		TokenID(uuid.New()),
		TokenHash("next-hash"),
		absoluteExpiresAt.Add(time.Minute),
		nil,
		nil,
		now.Add(5*time.Minute),
	)
	require.Error(t, err)
	require.Len(t, entity.Tokens, 1)
	require.Nil(t, entity.Tokens[0].RotatedAt)
	require.Nil(t, entity.LastRotatedAt)
}

func TestRevokeRefreshTokenFamilyRevokesEveryToken(t *testing.T) {
	t.Parallel()

	now := refreshTokenTestTime()
	initialTokenID := TokenID(uuid.New())
	entity := newRefreshTokenFamilyForTest(t, initialTokenID, now, now.Add(time.Hour), now.Add(24*time.Hour))
	_, err := entity.Rotate(
		initialTokenID,
		TokenID(uuid.New()),
		TokenHash("next-hash"),
		now.Add(2*time.Hour),
		nil,
		nil,
		now.Add(5*time.Minute),
	)
	require.NoError(t, err)

	revokedAt := now.Add(10 * time.Minute)
	require.NoError(t, entity.Revoke("session_revoked", revokedAt))
	require.Equal(t, revokedAt, *entity.RevokedAt)
	for _, token := range entity.Tokens {
		require.Equal(t, revokedAt, *token.RevokedAt)
		require.Equal(t, "session_revoked", *token.RevokeReason)
	}
}

func newRefreshTokenFamilyForTest(
	t *testing.T,
	initialTokenID TokenID,
	now time.Time,
	tokenExpiresAt time.Time,
	absoluteExpiresAt time.Time,
) *RefreshTokenFamily {
	t.Helper()

	entity, err := New(
		ID(uuid.New()),
		sessiondomain.ID(uuid.New()),
		initialTokenID,
		TokenHash("initial-hash"),
		tokenExpiresAt,
		absoluteExpiresAt,
		nil,
		nil,
		now,
	)
	require.NoError(t, err)
	return entity
}

func refreshTokenTestTime() time.Time {
	return time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
}
