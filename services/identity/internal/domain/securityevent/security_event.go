package securityevent

import (
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

type SecurityEvent struct {
	ID ID

	Type   EventType
	Result Result
	Actor  Actor

	SessionID     *sessiondomain.ID
	OAuthClientID *oauthclientdomain.ID

	IdentifierHash IdentifierHash
	IPAddress      *sessiondomain.IPAddress
	UserAgent      *string
	CorrelationID  *uuid.UUID
	Metadata       Metadata

	OccurredAt time.Time
}

func New(
	id ID,
	eventType EventType,
	result Result,
	actor Actor,
	sessionID *sessiondomain.ID,
	oauthClientID *oauthclientdomain.ID,
	identifierHash IdentifierHash,
	ipAddress *sessiondomain.IPAddress,
	userAgent *string,
	correlationID *uuid.UUID,
	metadata Metadata,
	occurredAt time.Time,
) (*SecurityEvent, error) {
	if !id.IsValid() {
		return nil, errors.New("create security event: identifier is required")
	}
	if sessionID != nil && !sessionID.IsValid() {
		return nil, errors.New("create security event: session identifier is invalid")
	}
	if oauthClientID != nil && !oauthClientID.IsValid() {
		return nil, errors.New("create security event: OAuth client identifier is invalid")
	}
	if correlationID != nil && *correlationID == uuid.Nil {
		return nil, errors.New("create security event: correlation identifier is invalid")
	}
	if occurredAt.IsZero() {
		return nil, errors.New("create security event: occurrence time is invalid")
	}

	return &SecurityEvent{
		ID: id,

		Type:   eventType,
		Result: result,
		Actor:  copyActor(actor),

		SessionID:     domain.ClonePointer(sessionID),
		OAuthClientID: domain.ClonePointer(oauthClientID),

		IdentifierHash: slices.Clone(identifierHash),
		IPAddress:      domain.ClonePointer(ipAddress),
		UserAgent:      domain.NormalizeOptionalText(userAgent),
		CorrelationID:  domain.ClonePointer(correlationID),
		Metadata:       slices.Clone(metadata),

		OccurredAt: occurredAt,
	}, nil
}
