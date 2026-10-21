package emailverificationchallenge

import (
	"errors"
	"fmt"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

type EmailVerificationChallenge struct {
	ID ID

	EmailID identitydomain.EmailID

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
	emailID identitydomain.EmailID,
	tokenHash TokenHash,
	maxAttempts int,
	expiresAt time.Time,
	requestedFromIP *sessiondomain.IPAddress,
	now time.Time,
) (*EmailVerificationChallenge, error) {
	if !id.IsValid() {
		return nil, errors.New("create email verification challenge: identifier is required")
	}
	if !emailID.IsValid() {
		return nil, errors.New("create email verification challenge: email identifier is invalid")
	}
	if maxAttempts <= 0 {
		return nil, errors.New("create email verification challenge: max attempts must be positive")
	}
	if now.IsZero() || !expiresAt.After(now) {
		return nil, errors.New("create email verification challenge: expiration is invalid")
	}

	return &EmailVerificationChallenge{
		ID: id,

		EmailID: emailID,

		TokenHash:   tokenHash,
		MaxAttempts: maxAttempts,

		RequestedFromIP: domain.ClonePointer(requestedFromIP),

		CreatedAt: now,
		ExpiresAt: expiresAt,
	}, nil
}

func (c *EmailVerificationChallenge) RecordFailedAttempt(now time.Time) (AttemptOutcome, error) {
	if err := c.ensureAttemptAllowed(now); err != nil {
		return "", fmt.Errorf("record failed email verification attempt: %w", err)
	}

	c.Attempts++
	c.LastAttemptAt = domain.ClonePointer(&now)
	if c.Attempts == c.MaxAttempts {
		return AttemptOutcomeExhausted, nil
	}
	return AttemptOutcomeRecorded, nil
}

func (c *EmailVerificationChallenge) Consume(consumedFromIP *sessiondomain.IPAddress, now time.Time) error {
	if err := c.ensureAttemptAllowed(now); err != nil {
		return fmt.Errorf("consume email verification challenge: %w", err)
	}

	c.Attempts++
	c.LastAttemptAt = domain.ClonePointer(&now)
	c.ConsumedAt = domain.ClonePointer(&now)
	c.ConsumedFromIP = domain.ClonePointer(consumedFromIP)
	return nil
}

func (c *EmailVerificationChallenge) Revoke(now time.Time) error {
	if c == nil {
		return errors.New("revoke email verification challenge: challenge is nil")
	}
	if c.RevokedAt != nil {
		return nil
	}
	if c.ConsumedAt != nil {
		return fmt.Errorf("revoke email verification challenge: %w", ErrConsumed)
	}
	if now.IsZero() || now.Before(c.latestChangedAt()) {
		return errors.New("revoke email verification challenge: time is invalid")
	}

	c.RevokedAt = domain.ClonePointer(&now)
	return nil
}

func (c *EmailVerificationChallenge) ensureAttemptAllowed(now time.Time) error {
	if c == nil {
		return errors.New("validate email verification challenge for attempt: challenge is nil")
	}
	if now.IsZero() || now.Before(c.latestChangedAt()) {
		return errors.New("validate email verification challenge for attempt: time is invalid")
	}
	if c.ConsumedAt != nil {
		return fmt.Errorf("validate email verification challenge for attempt: %w", ErrConsumed)
	}
	if c.RevokedAt != nil {
		return fmt.Errorf("validate email verification challenge for attempt: %w", ErrRevoked)
	}
	if !now.Before(c.ExpiresAt) {
		return fmt.Errorf("validate email verification challenge for attempt: %w", ErrExpired)
	}
	if c.Attempts >= c.MaxAttempts {
		return fmt.Errorf("validate email verification challenge for attempt: %w", ErrAttemptsExhausted)
	}

	return nil
}

func (c *EmailVerificationChallenge) latestChangedAt() time.Time {
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

func (c *EmailVerificationChallenge) latestAttemptAt() time.Time {
	if c.LastAttemptAt != nil && c.LastAttemptAt.After(c.CreatedAt) {
		return *c.LastAttemptAt
	}
	return c.CreatedAt
}
