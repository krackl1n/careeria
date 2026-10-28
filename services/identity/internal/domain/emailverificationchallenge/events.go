package emailverificationchallenge

import (
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/event"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

const AggregateType event.AggregateType = "email_verification_challenge"

const EventTypeIssued event.Type = "EmailVerificationChallengeIssued"

type IssuedEvent struct {
	ChallengeID uuid.UUID                   `json:"challenge_id"`
	EmailID     uuid.UUID                   `json:"email_id"`
	Email       identitydomain.EmailAddress `json:"email"`
	Code        string                      `json:"code"`

	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
