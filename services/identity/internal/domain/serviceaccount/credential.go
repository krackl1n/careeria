package serviceaccount

import (
	"errors"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

type Credential struct {
	ID   CredentialID
	Type CredentialType

	SecretHash   SecretHash
	SecretPrefix *SecretPrefix
	Name         *string

	CreatedByIdentityID *identitydomain.ID

	CreatedAt    time.Time
	UpdatedAt    time.Time
	ExpiresAt    *time.Time
	RevokedAt    *time.Time
	RevokeReason *string
	LastUsedAt   *time.Time
}

func newCredential(
	id CredentialID,
	secretHash SecretHash,
	secretPrefix *SecretPrefix,
	name *string,
	createdByIdentityID *identitydomain.ID,
	expiresAt *time.Time,
	now time.Time,
) (Credential, error) {
	if !id.IsValid() {
		return Credential{}, errors.New("credential identifier is required")
	}
	if createdByIdentityID != nil && !createdByIdentityID.IsValid() {
		return Credential{}, errors.New("credential creator identifier is invalid")
	}
	if now.IsZero() {
		return Credential{}, errors.New("credential creation time is invalid")
	}
	if expiresAt != nil && !expiresAt.After(now) {
		return Credential{}, errors.New("credential expiration must follow creation")
	}

	return Credential{
		ID:   id,
		Type: CredentialTypeSharedSecret,

		SecretHash:   secretHash,
		SecretPrefix: domain.ClonePointer(secretPrefix),
		Name:         domain.NormalizeOptionalText(name),

		CreatedByIdentityID: domain.ClonePointer(createdByIdentityID),

		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: domain.ClonePointer(expiresAt),
	}, nil
}

func (c *Credential) Rename(name *string, now time.Time) (bool, error) {
	if err := c.ensureChangeAllowed(now); err != nil {
		return false, err
	}
	normalizedName := domain.NormalizeOptionalText(name)
	if domain.EqualOptionalText(c.Name, normalizedName) {
		return false, nil
	}

	c.Name = normalizedName
	c.UpdatedAt = now
	return true, nil
}

func (c *Credential) Revoke(now time.Time, reason string) error {
	if c == nil {
		return errors.New("credential is nil")
	}
	if c.RevokedAt != nil {
		return ErrCredentialRevoked
	}
	if err := c.ensureChangeAllowed(now); err != nil {
		return err
	}
	normalizedReason, ok := domain.NormalizeRequiredText(reason)
	if !ok {
		return errors.New("reason is required")
	}

	c.RevokedAt = domain.ClonePointer(&now)
	c.RevokeReason = &normalizedReason
	c.UpdatedAt = now
	return nil
}

func (c *Credential) Validate(now time.Time) error {
	if c == nil {
		return errors.New("credential is nil")
	}
	if now.IsZero() || now.Before(c.UpdatedAt) {
		return errors.New("credential use time is invalid")
	}
	if c.RevokedAt != nil {
		return ErrCredentialRevoked
	}
	if c.ExpiresAt != nil && !now.Before(*c.ExpiresAt) {
		return ErrCredentialExpired
	}

	return nil
}

func (c *Credential) RecordUsed(now time.Time) error {
	if err := c.Validate(now); err != nil {
		return err
	}

	c.LastUsedAt = domain.ClonePointer(&now)
	c.UpdatedAt = now
	return nil
}

func (c *Credential) IsUsable(now time.Time) bool {
	return c != nil && c.RevokedAt == nil && (c.ExpiresAt == nil || now.Before(*c.ExpiresAt))
}

func (c *Credential) ensureChangeAllowed(now time.Time) error {
	if c == nil {
		return errors.New("credential is nil")
	}
	if now.IsZero() || now.Before(c.UpdatedAt) {
		return errors.New("credential change time is invalid")
	}

	return nil
}
