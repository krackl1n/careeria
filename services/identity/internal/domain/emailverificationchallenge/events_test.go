package emailverificationchallenge

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIssuedEventPayload(t *testing.T) {
	t.Parallel()

	payload := IssuedEvent{
		ChallengeID: uuid.New(),
		EmailID:     uuid.New(),
		Email:       "user@example.com",
		Code:        "verification-code",
		IssuedAt:    emailVerificationChallengeTestTime(),
		ExpiresAt:   emailVerificationChallengeTestTime().Add(10 * time.Minute),
	}

	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"challenge_id":"`+payload.ChallengeID.String()+`",
		"email_id":"`+payload.EmailID.String()+`",
		"email":"user@example.com",
		"code":"verification-code",
		"issued_at":"2026-07-31T12:00:00Z",
		"expires_at":"2026-07-31T12:10:00Z"
	}`, string(encoded))
}

func TestEmailVerificationChallengeEventTypeIsStable(t *testing.T) {
	t.Parallel()

	require.Equal(t, "email_verification_challenge", string(AggregateType))
	require.Equal(t, "EmailVerificationChallengeIssued", string(EventTypeIssued))
}
