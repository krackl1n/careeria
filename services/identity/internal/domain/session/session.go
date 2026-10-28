package session

import (
	"errors"
	"fmt"
	"slices"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
)

type Session struct {
	ID            ID
	IdentityID    identitydomain.ID
	OAuthClientID oauthclientdomain.ID

	AuthenticationLevel  authenticationdomain.Level
	AuthenticatedMethods []authenticationdomain.Method

	Scopes []oauthdomain.Scope

	IPAddress  *IPAddress
	UserAgent  *string
	DeviceName *string

	CreatedAt       time.Time
	AuthenticatedAt time.Time
	LastSeenAt      *time.Time

	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time

	RevokedAt    *time.Time
	RevokeReason *string
}

func New(
	id ID,
	identityID identitydomain.ID,
	oauthClientID oauthclientdomain.ID,
	authenticationLevel authenticationdomain.Level,
	authenticatedMethods []authenticationdomain.Method,
	scopes []oauthdomain.Scope,
	ipAddress *IPAddress,
	userAgent *string,
	deviceName *string,
	authenticatedAt time.Time,
	idleExpiresAt time.Time,
	absoluteExpiresAt time.Time,
	now time.Time,
) (*Session, error) {
	if !id.IsValid() {
		return nil, errors.New("create session: identifier is required")
	}
	if !identityID.IsValid() {
		return nil, errors.New("create session: identity identifier is invalid")
	}
	if !oauthClientID.IsValid() {
		return nil, errors.New("create session: OAuth client identifier is invalid")
	}
	if now.IsZero() {
		return nil, errors.New("create session: time is invalid")
	}
	if authenticatedAt.IsZero() || authenticatedAt.After(now) {
		return nil, errors.New("create session: authentication time is invalid")
	}
	resolvedLevel, err := authenticationdomain.ResolveLevel(authenticatedMethods)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	if !authenticationLevel.IsValid() || authenticationLevel != resolvedLevel {
		return nil, errors.New("create session: authentication level does not match methods")
	}
	if domain.HasDuplicates(scopes) {
		return nil, errors.New("create session: scope is duplicated")
	}
	if !idleExpiresAt.After(now) || !absoluteExpiresAt.After(now) || idleExpiresAt.After(absoluteExpiresAt) {
		return nil, errors.New("create session: expiration is invalid")
	}

	return &Session{
		ID:                   id,
		IdentityID:           identityID,
		OAuthClientID:        oauthClientID,
		AuthenticationLevel:  authenticationLevel,
		AuthenticatedMethods: slices.Clone(authenticatedMethods),
		Scopes:               slices.Clone(scopes),
		IPAddress:            domain.ClonePointer(ipAddress),
		UserAgent:            domain.NormalizeOptionalText(userAgent),
		DeviceName:           domain.NormalizeOptionalText(deviceName),
		CreatedAt:            now,
		AuthenticatedAt:      authenticatedAt,
		IdleExpiresAt:        idleExpiresAt,
		AbsoluteExpiresAt:    absoluteExpiresAt,
	}, nil
}

func (s *Session) ValidateForUse(now time.Time) error {
	if s == nil {
		return errors.New("validate session for use: session is nil")
	}
	if now.IsZero() || now.Before(s.latestChangedAt()) {
		return errors.New("validate session for use: time is invalid")
	}
	if s.RevokedAt != nil {
		return fmt.Errorf("validate session for use: %w", ErrRevoked)
	}
	if !now.Before(s.AbsoluteExpiresAt) {
		return fmt.Errorf("validate session for use: %w", ErrAbsoluteExpired)
	}
	if !now.Before(s.IdleExpiresAt) {
		return fmt.Errorf("validate session for use: %w", ErrIdleExpired)
	}

	return nil
}

func (s *Session) RecordActivity(now, idleExpiresAt time.Time) error {
	if err := s.ValidateForUse(now); err != nil {
		return fmt.Errorf("record session activity: %w", err)
	}
	if !idleExpiresAt.After(now) || idleExpiresAt.After(s.AbsoluteExpiresAt) {
		return errors.New("record session activity: idle expiration is invalid")
	}

	s.LastSeenAt = domain.ClonePointer(&now)
	if idleExpiresAt.After(s.IdleExpiresAt) {
		s.IdleExpiresAt = idleExpiresAt
	}
	return nil
}

func (s *Session) Revoke(reason string, now time.Time) error {
	if s == nil {
		return errors.New("revoke session: session is nil")
	}
	if s.RevokedAt != nil {
		return nil
	}
	if now.IsZero() || now.Before(s.latestChangedAt()) {
		return errors.New("revoke session: time is invalid")
	}
	var ok bool
	reason, ok = domain.NormalizeRequiredText(reason)
	if !ok {
		return errors.New("revoke session: reason is required")
	}

	s.RevokedAt = domain.ClonePointer(&now)
	s.RevokeReason = &reason
	return nil
}

func (s *Session) latestChangedAt() time.Time {
	if s == nil {
		return time.Time{}
	}
	latest := s.AuthenticatedAt
	for _, value := range []*time.Time{s.LastSeenAt, s.RevokedAt} {
		if value != nil && value.After(latest) {
			latest = *value
		}
	}
	return latest
}
