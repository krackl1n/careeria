package phoneverificationchallenge

import (
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/event"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

const AggregateType event.AggregateType = "phone_verification_challenge"

const EventTypeIssued event.Type = "PhoneVerificationChallengeIssued"

type IssuedEvent struct {
	ChallengeID uuid.UUID                  `json:"challenge_id"`
	PhoneID     uuid.UUID                  `json:"phone_id"`
	Phone       identitydomain.PhoneNumber `json:"phone"`
	Code        string                     `json:"code"`

	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
