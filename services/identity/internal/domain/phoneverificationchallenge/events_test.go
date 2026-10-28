package phoneverificationchallenge

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
		PhoneID:     uuid.New(),
		Phone:       "+79991234567",
		Code:        "verification-code",
		IssuedAt:    phoneVerificationChallengeTestTime(),
		ExpiresAt:   phoneVerificationChallengeTestTime().Add(10 * time.Minute),
	}

	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"challenge_id":"`+payload.ChallengeID.String()+`",
		"phone_id":"`+payload.PhoneID.String()+`",
		"phone":"+79991234567",
		"code":"verification-code",
		"issued_at":"2026-07-31T12:00:00Z",
		"expires_at":"2026-07-31T12:10:00Z"
	}`, string(encoded))
}

func TestPhoneVerificationChallengeEventTypeIsStable(t *testing.T) {
	t.Parallel()

	require.Equal(t, "phone_verification_challenge", string(AggregateType))
	require.Equal(t, "PhoneVerificationChallengeIssued", string(EventTypeIssued))
}
