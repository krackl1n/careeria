package recoverycodeset

import (
	"errors"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type RecoveryCode struct {
	ID   CodeID
	Hash CodeHash

	CreatedAt time.Time
	UsedAt    *time.Time
}

func newRecoveryCode(id CodeID, hash CodeHash, now time.Time) (RecoveryCode, error) {
	if !id.IsValid() {
		return RecoveryCode{}, errors.New("code identifier is invalid")
	}
	if now.IsZero() {
		return RecoveryCode{}, errors.New("code creation time is invalid")
	}

	return RecoveryCode{
		ID:        id,
		Hash:      hash,
		CreatedAt: now,
	}, nil
}

func (c *RecoveryCode) Use(now time.Time) error {
	if c == nil {
		return errors.New("recovery code is nil")
	}
	if now.IsZero() || now.Before(c.ChangedAt()) {
		return errors.New("recovery code use time is invalid")
	}
	if c.UsedAt != nil {
		return ErrCodeUsed
	}

	c.UsedAt = domain.ClonePointer(&now)
	return nil
}

func (c *RecoveryCode) IsUsed() bool {
	return c != nil && c.UsedAt != nil
}

func (c *RecoveryCode) ChangedAt() time.Time {
	if c == nil {
		return time.Time{}
	}
	if c.UsedAt != nil && c.UsedAt.After(c.CreatedAt) {
		return *c.UsedAt
	}
	return c.CreatedAt
}
