package passwordresetchallenge

import (
	"errors"
	"fmt"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

type PasswordResetChallenge struct {
	ID ID

	IdentityID identitydomain.ID

	TokenHash   TokenHash
	Attempts    int
	MaxAttempts int

	RequestedFromIP *sessiondomain.IPAddress
	ConsumedFromIP  *sessiondomain.IPAddress

	CreatedAt     time.Time
	ExpiresAt     time.Time
	LastAttemptAt *time.Time
	ConsumedAt    *time.Time
	RevokedAt     *time.Time
}

func New(
	id ID,
	identityID identitydomain.ID,
	tokenHash TokenHash,
	maxAttempts int,
	expiresAt time.Time,
	requestedFromIP *sessiondomain.IPAddress,
	now time.Time,
) (*PasswordResetChallenge, error) {
	if !id.IsValid() {
		return nil, errors.New("create password reset challenge: identifier is required")
	}
	if !identityID.IsValid() {
		return nil, errors.New("create password reset challenge: identity identifier is invalid")
	}
	if maxAttempts <= 0 {
		return nil, errors.New("create password reset challenge: max attempts must be positive")
	}
	if now.IsZero() || !expiresAt.After(now) {
		return nil, errors.New("create password reset challenge: expiration is invalid")
	}

	return &PasswordResetChallenge{
		ID: id,

		IdentityID: identityID,

		TokenHash:   tokenHash,
		MaxAttempts: maxAttempts,

		RequestedFromIP: domain.ClonePointer(requestedFromIP),

		CreatedAt: now,
		ExpiresAt: expiresAt,
	}, nil
}

func (c *PasswordResetChallenge) RecordFailedAttempt(now time.Time) (AttemptOutcome, error) {
	if err := c.ensureAttemptAllowed(now); err != nil {
		return "", fmt.Errorf("record failed password reset attempt: %w", err)
	}

	c.Attempts++
	c.LastAttemptAt = domain.ClonePointer(&now)
	if c.Attempts == c.MaxAttempts {
		return AttemptOutcomeExhausted, nil
	}

	return AttemptOutcomeRecorded, nil
}

func (c *PasswordResetChallenge) Consume(consumedFromIP *sessiondomain.IPAddress, now time.Time) error {
	if err := c.ensureAttemptAllowed(now); err != nil {
		return fmt.Errorf("consume password reset challenge: %w", err)
	}

	c.Attempts++
	c.LastAttemptAt = domain.ClonePointer(&now)
	c.ConsumedAt = domain.ClonePointer(&now)
	c.ConsumedFromIP = domain.ClonePointer(consumedFromIP)
	return nil
}

func (c *PasswordResetChallenge) Revoke(now time.Time) error {
	if c == nil {
		return errors.New("revoke password reset challenge: challenge is nil")
	}
	if c.RevokedAt != nil {
		return nil
	}
	if c.ConsumedAt != nil {
		return fmt.Errorf("revoke password reset challenge: %w", ErrConsumed)
	}
	if now.IsZero() || now.Before(c.latestChangedAt()) {
		return errors.New("revoke password reset challenge: time is invalid")
	}

	c.RevokedAt = domain.ClonePointer(&now)
	return nil
}

func (c *PasswordResetChallenge) ensureAttemptAllowed(now time.Time) error {
	if c == nil {
		return errors.New("validate password reset challenge for attempt: challenge is nil")
	}
	if now.IsZero() || now.Before(c.latestChangedAt()) {
		return errors.New("validate password reset challenge for attempt: time is invalid")
	}
	if c.ConsumedAt != nil {
		return fmt.Errorf("validate password reset challenge for attempt: %w", ErrConsumed)
	}
	if c.RevokedAt != nil {
		return fmt.Errorf("validate password reset challenge for attempt: %w", ErrRevoked)
	}
	if !now.Before(c.ExpiresAt) {
		return fmt.Errorf("validate password reset challenge for attempt: %w", ErrExpired)
	}
	if c.Attempts >= c.MaxAttempts {
		return fmt.Errorf("validate password reset challenge for attempt: %w", ErrAttemptsExhausted)
	}

	return nil
}

func (c *PasswordResetChallenge) latestChangedAt() time.Time {
	if c == nil {
		return time.Time{}
	}
	latest := c.latestAttemptAt()
	for _, value := range []*time.Time{c.ConsumedAt, c.RevokedAt} {
		if value != nil && value.After(latest) {
			latest = *value
		}
	}

	return latest
}

func (c *PasswordResetChallenge) latestAttemptAt() time.Time {
	if c.LastAttemptAt != nil && c.LastAttemptAt.After(c.CreatedAt) {
		return *c.LastAttemptAt
	}

	return c.CreatedAt
}
