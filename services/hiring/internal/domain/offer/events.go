package offer

import (
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/hiring/internal/domain/event"
)

const (
	AggregateTypeJobOffer event.AggregateType = "job_offer"

	EventTypeOfferCreated  event.Type = "hiring.offer.created"
	EventTypeOfferSent     event.Type = "hiring.offer.sent"
	EventTypeOfferAccepted event.Type = "hiring.offer.accepted"
	EventTypeOfferDeclined event.Type = "hiring.offer.declined"
	EventTypeOfferRevoked  event.Type = "hiring.offer.revoked"
	EventTypeOfferExpired  event.Type = "hiring.offer.expired"
)

type OfferCreatedPayload struct {
	OfferID       uuid.UUID `json:"offer_id"`
	ApplicationID uuid.UUID `json:"application_id"`
	Compensation  int64     `json:"compensation"`
	Currency      string    `json:"currency"`
	Grade         string    `json:"grade"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type OfferSentPayload struct {
	OfferID   uuid.UUID `json:"offer_id"`
	SentAt    time.Time `json:"sent_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type OfferAcceptedPayload struct {
	OfferID    uuid.UUID `json:"offer_id"`
	AcceptedAt time.Time `json:"accepted_at"`
}

type OfferDeclinedPayload struct {
	OfferID    uuid.UUID `json:"offer_id"`
	Reason     string    `json:"reason,omitempty"`
	DeclinedAt time.Time `json:"declined_at"`
}

type OfferRevokedPayload struct {
	OfferID   uuid.UUID `json:"offer_id"`
	Reason    string    `json:"reason,omitempty"`
	RevokedAt time.Time `json:"revoked_at"`
}

type OfferExpiredPayload struct {
	OfferID   uuid.UUID `json:"offer_id"`
	ExpiredAt time.Time `json:"expired_at"`
}

