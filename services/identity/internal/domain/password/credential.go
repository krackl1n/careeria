package password

import (
	"errors"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type Credential struct {
	Hash       Hash
	MustChange bool

	PasswordUpdatedAt time.Time
	ExpiresAt         *time.Time
	CompromisedAt     *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func newCredential(hash Hash, mustChange bool, expiresAt *time.Time, now time.Time) (Credential, error) {
	if now.IsZero() {
		return Credential{}, errors.New("time is invalid")
	}
	if expiresAt != nil && !expiresAt.After(now) {
		return Credential{}, errors.New("expiration must follow creation")
	}

	return Credential{
		Hash:              hash,
		MustChange:        mustChange,
		PasswordUpdatedAt: now,
		ExpiresAt:         domain.ClonePointer(expiresAt),
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

func (c *Credential) Change(hash Hash, mustChange bool, expiresAt *time.Time, now time.Time) error {
	if err := c.ensureChangeAllowed(now); err != nil {
		return err
	}
	if expiresAt != nil && !expiresAt.After(now) {
		return errors.New("expiration must follow password update")
	}

	c.Hash = hash
	c.MustChange = mustChange
	c.PasswordUpdatedAt = now
	c.ExpiresAt = domain.ClonePointer(expiresAt)
	c.CompromisedAt = nil
	c.UpdatedAt = now
	return nil
}

func (c *Credential) RequireChange(now time.Time) (bool, error) {
	if err := c.ensureChangeAllowed(now); err != nil {
		return false, err
	}
	if c.MustChange {
		return false, nil
	}

	c.MustChange = true
	c.UpdatedAt = now
	return true, nil
}

func (c *Credential) MarkCompromised(now time.Time) (bool, error) {
	if err := c.ensureChangeAllowed(now); err != nil {
		return false, err
	}
	if c.CompromisedAt != nil {
		return false, nil
	}

	c.MustChange = true
	c.CompromisedAt = domain.ClonePointer(&now)
	c.UpdatedAt = now
	return true, nil
}

func (c *Credential) ValidateForAuthentication(now time.Time) error {
	if c == nil {
		return errors.New("password credential is nil")
	}
	if now.IsZero() || now.Before(c.UpdatedAt) {
		return errors.New("password credential use time is invalid")
	}
	if c.CompromisedAt != nil {
		return ErrCompromised
	}
	if c.ExpiresAt != nil && !now.Before(*c.ExpiresAt) {
		return ErrExpired
	}

	return nil
}

func (c *Credential) ChangedAt() time.Time {
	if c == nil {
		return time.Time{}
	}
	return c.UpdatedAt
}

func (c *Credential) ensureChangeAllowed(now time.Time) error {
	if c == nil {
		return errors.New("password credential is nil")
	}
	if now.IsZero() || now.Before(c.UpdatedAt) {
		return errors.New("password credential change time is invalid")
	}

	return nil
}
