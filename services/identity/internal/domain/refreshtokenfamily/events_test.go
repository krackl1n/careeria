package refreshtokenfamily

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRefreshTokenEventsDoNotExposeTokenHash(t *testing.T) {
	t.Parallel()

	payloads := []any{
		CreatedEvent{
			FamilyID:              uuid.New(),
			SessionID:             uuid.New(),
			InitialTokenID:        uuid.New(),
			InitialTokenExpiresAt: refreshTokenTestTime().Add(time.Hour),
			AbsoluteExpiresAt:     refreshTokenTestTime().Add(24 * time.Hour),
			CreatedAt:             refreshTokenTestTime(),
		},
		TokenRotatedEvent{
			FamilyID:          uuid.New(),
			SessionID:         uuid.New(),
			PreviousTokenID:   uuid.New(),
			NewTokenID:        uuid.New(),
			NewTokenExpiresAt: refreshTokenTestTime().Add(time.Hour),
			RotatedAt:         refreshTokenTestTime(),
		},
	}

	for _, payload := range payloads {
		encoded, err := json.Marshal(payload)
		require.NoError(t, err)
		require.NotContains(t, strings.ToLower(string(encoded)), "hash")
		require.NotContains(t, strings.ToLower(string(encoded)), "secret")
	}
}
