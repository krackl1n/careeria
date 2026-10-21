package offer

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestJobOfferLifecycle(t *testing.T) {
	now := time.Now().UTC()
	offerID := uuid.New()
	applicationID := uuid.New()
	expiresAt := now.Add(48 * time.Hour)

	// Invariant: min 24 hours duration
	_, err := Create(offerID, applicationID, 250000, "RUB", "Senior Go", now.Add(12*time.Hour), now)
	require.ErrorIs(t, err, ErrInvalidExpiration)

	// Create valid offer
	of, err := Create(offerID, applicationID, 250000, "RUB", "Senior Go", expiresAt, now)
	require.NoError(t, err)
	require.Equal(t, StatusDraft, of.Status)
	require.Equal(t, int64(1), of.Version)

	// Send offer
	err = of.Send(now)
	require.NoError(t, err)
	require.Equal(t, StatusSent, of.Status)

	// Cannot accept an already expired offer
	expiredNow := expiresAt.Add(time.Minute)
	err = of.Accept(expiredNow)
	require.ErrorIs(t, err, ErrOfferExpired)
	require.Equal(t, StatusExpired, of.Status)
}

func TestJobOfferAcceptAndDecline(t *testing.T) {
	now := time.Now().UTC()
	expiresAt := now.Add(48 * time.Hour)

	t.Run("accept offer within validity period", func(t *testing.T) {
		of, err := Create(uuid.New(), uuid.New(), 300000, "RUB", "Lead Go", expiresAt, now)
		require.NoError(t, err)

		err = of.Send(now)
		require.NoError(t, err)

		err = of.Accept(now.Add(2 * time.Hour))
		require.NoError(t, err)
		require.Equal(t, StatusAccepted, of.Status)
	})

	t.Run("decline offer", func(t *testing.T) {
		of, err := Create(uuid.New(), uuid.New(), 300000, "RUB", "Lead Go", expiresAt, now)
		require.NoError(t, err)

		err = of.Send(now)
		require.NoError(t, err)

		err = of.Decline("Candidate chose other company", now.Add(2*time.Hour))
		require.NoError(t, err)
		require.Equal(t, StatusDeclined, of.Status)
	})

	t.Run("revoke offer", func(t *testing.T) {
		of, err := Create(uuid.New(), uuid.New(), 300000, "RUB", "Lead Go", expiresAt, now)
		require.NoError(t, err)

		err = of.Revoke("Budget frozen", now)
		require.NoError(t, err)
		require.Equal(t, StatusRevoked, of.Status)
	})
}
