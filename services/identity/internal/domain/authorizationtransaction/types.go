package authorizationtransaction

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ID uuid.UUID

func (id ID) UUID() uuid.UUID { return uuid.UUID(id) }

func (id ID) IsValid() bool { return uuid.UUID(id) != uuid.Nil }

type Status string

const (
	StatusPending   Status = "pending"
	StatusReady     Status = "ready"
	StatusCompleted Status = "completed"
	StatusDenied    Status = "denied"
)

func ParseStatus(value string) (Status, error) {
	status := Status(value)
	switch status {
	case StatusPending, StatusReady, StatusCompleted, StatusDenied:
		return status, nil
	default:
		return "", errors.New("unknown authorization transaction status")
	}
}

type AuthorizationSource string

const (
	AuthorizationSourceUserConsent     AuthorizationSource = "user_consent"
	AuthorizationSourceRememberedGrant AuthorizationSource = "remembered_grant"
	AuthorizationSourceTrustedClient   AuthorizationSource = "trusted_client"
)

func ParseAuthorizationSource(value string) (AuthorizationSource, error) {
	source := AuthorizationSource(value)
	switch source {
	case AuthorizationSourceUserConsent,
		AuthorizationSourceRememberedGrant,
		AuthorizationSourceTrustedClient:
		return source, nil
	default:
		return "", errors.New("unknown authorization source")
	}
}

type VerificationChannel string

const (
	VerificationChannelEmail VerificationChannel = "email"
	VerificationChannelPhone VerificationChannel = "phone"
)

func ParseVerificationChannel(value string) (VerificationChannel, error) {
	channel := VerificationChannel(value)
	switch channel {
	case VerificationChannelEmail, VerificationChannelPhone:
		return channel, nil
	default:
		return "", errors.New("unknown verification channel")
	}
}

type VerificationReference struct {
	Channel     VerificationChannel
	ContactID   uuid.UUID
	ChallengeID uuid.UUID
	ExpiresAt   time.Time
}

func NewVerificationReference(
	channel VerificationChannel,
	contactID uuid.UUID,
	challengeID uuid.UUID,
	expiresAt time.Time,
) (VerificationReference, error) {
	if _, err := ParseVerificationChannel(string(channel)); err != nil {
		return VerificationReference{}, fmt.Errorf("create verification reference: %w", err)
	}
	if contactID == uuid.Nil {
		return VerificationReference{}, errors.New("create verification reference: contact identifier is required")
	}
	if challengeID == uuid.Nil {
		return VerificationReference{}, errors.New("create verification reference: challenge identifier is required")
	}
	if expiresAt.IsZero() {
		return VerificationReference{}, errors.New("create verification reference: expiration is required")
	}

	return VerificationReference{
		Channel:     channel,
		ContactID:   contactID,
		ChallengeID: challengeID,
		ExpiresAt:   expiresAt,
	}, nil
}

type State string

const maxStateLength = 2048

func NewState(value string) (State, error) {
	if value == "" {
		return "", errors.New("OAuth state is required")
	}
	if len(value) > maxStateLength {
		return "", errors.New("OAuth state exceeds 2048 bytes")
	}

	return State(value), nil
}
