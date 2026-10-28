package refreshtokenfamily

import (
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/event"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

const AggregateType event.AggregateType = "refresh_token_family"

const (
	EventTypeCreated       event.Type = "RefreshTokenFamilyCreated"
	EventTypeTokenRotated  event.Type = "RefreshTokenRotated"
	EventTypeReuseDetected event.Type = "RefreshTokenReuseDetected"
	EventTypeRevoked       event.Type = "RefreshTokenFamilyRevoked"
)

type CreatedEvent struct {
	FamilyID              uuid.UUID                `json:"family_id"`
	SessionID             uuid.UUID                `json:"session_id"`
	InitialTokenID        uuid.UUID                `json:"initial_token_id"`
	InitialTokenExpiresAt time.Time                `json:"initial_token_expires_at"`
	AbsoluteExpiresAt     time.Time                `json:"absolute_expires_at"`
	CreatedFromIP         *sessiondomain.IPAddress `json:"created_from_ip,omitempty"`
	CreatedUserAgent      *string                  `json:"created_user_agent,omitempty"`
	CreatedAt             time.Time                `json:"created_at"`
}

type TokenRotatedEvent struct {
	FamilyID          uuid.UUID                `json:"family_id"`
	SessionID         uuid.UUID                `json:"session_id"`
	PreviousTokenID   uuid.UUID                `json:"previous_token_id"`
	NewTokenID        uuid.UUID                `json:"new_token_id"`
	NewTokenExpiresAt time.Time                `json:"new_token_expires_at"`
	CreatedFromIP     *sessiondomain.IPAddress `json:"created_from_ip,omitempty"`
	CreatedUserAgent  *string                  `json:"created_user_agent,omitempty"`
	RotatedAt         time.Time                `json:"rotated_at"`
}

type ReuseDetectedEvent struct {
	FamilyID      uuid.UUID `json:"family_id"`
	SessionID     uuid.UUID `json:"session_id"`
	ReusedTokenID uuid.UUID `json:"reused_token_id"`
	DetectedAt    time.Time `json:"detected_at"`
	RevokedAt     time.Time `json:"revoked_at"`
}

type RevokedEvent struct {
	FamilyID  uuid.UUID `json:"family_id"`
	SessionID uuid.UUID `json:"session_id"`
	Reason    string    `json:"reason"`
	RevokedAt time.Time `json:"revoked_at"`
}
