package session

import (
	"time"

	"github.com/google/uuid"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	"github.com/krackl1n/careeria/services/identity/internal/domain/event"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
)

const AggregateType event.AggregateType = "session"

const (
	EventTypeCreated event.Type = "SessionCreated"
	EventTypeRevoked event.Type = "SessionRevoked"
)

type CreatedEvent struct {
	SessionID            uuid.UUID                     `json:"session_id"`
	IdentityID           uuid.UUID                     `json:"identity_id"`
	OAuthClientID        uuid.UUID                     `json:"oauth_client_id"`
	AuthenticationLevel  authenticationdomain.Level    `json:"authentication_level"`
	AuthenticatedMethods []authenticationdomain.Method `json:"authenticated_methods"`
	Scopes               []oauthdomain.Scope           `json:"scopes"`
	IPAddress            *IPAddress                    `json:"ip_address,omitempty"`
	UserAgent            *string                       `json:"user_agent,omitempty"`
	DeviceName           *string                       `json:"device_name,omitempty"`
	AuthenticatedAt      time.Time                     `json:"authenticated_at"`
	IdleExpiresAt        time.Time                     `json:"idle_expires_at"`
	AbsoluteExpiresAt    time.Time                     `json:"absolute_expires_at"`
	CreatedAt            time.Time                     `json:"created_at"`
}

type RevokedEvent struct {
	SessionID     uuid.UUID `json:"session_id"`
	IdentityID    uuid.UUID `json:"identity_id"`
	OAuthClientID uuid.UUID `json:"oauth_client_id"`
	Reason        string    `json:"reason"`
	RevokedAt     time.Time `json:"revoked_at"`
}
