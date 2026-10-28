package identity

import (
	"errors"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type Email struct {
	ID      EmailID
	Address EmailAddress

	IsPrimary bool

	CreatedAt  time.Time
	UpdatedAt  time.Time
	VerifiedAt *time.Time
	DeletedAt  *time.Time
}

func newEmail(id EmailID, address EmailAddress, primary bool, now time.Time) (Email, error) {
	if !id.IsValid() || now.IsZero() {
		return Email{}, errors.New("identity email constructor arguments are invalid")
	}

	return Email{
		ID:        id,
		Address:   address,
		IsPrimary: primary,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (e *Email) Verify(now time.Time) error {
	if err := e.ensureChangeAllowed(now); err != nil {
		return err
	}
	if e.VerifiedAt != nil {
		return errors.New("email is already verified")
	}

	e.VerifiedAt = domain.ClonePointer(&now)
	e.UpdatedAt = now
	return nil
}

func (e *Email) Promote(now time.Time) (bool, error) {
	if err := e.ensureChangeAllowed(now); err != nil {
		return false, err
	}
	if e.VerifiedAt == nil {
		return false, errors.New("email is not verified")
	}
	if e.IsPrimary {
		return false, nil
	}

	e.IsPrimary = true
	e.UpdatedAt = now
	return true, nil
}

func (e *Email) Demote(now time.Time) error {
	if e == nil || !e.IsActive() || !e.IsPrimary {
		return nil
	}
	if err := e.ensureChangeAllowed(now); err != nil {
		return err
	}

	e.IsPrimary = false
	e.UpdatedAt = now
	return nil
}

func (e *Email) Remove(now time.Time) error {
	if err := e.ensureChangeAllowed(now); err != nil {
		return err
	}

	e.IsPrimary = false
	e.DeletedAt = domain.ClonePointer(&now)
	e.UpdatedAt = now
	return nil
}

func (e *Email) IsActive() bool {
	return e != nil && e.DeletedAt == nil
}

func (e *Email) IsUsable() bool {
	return e.IsActive() && e.VerifiedAt != nil
}

func (e *Email) IsPrimaryEmail() bool {
	return e.IsActive() && e.IsPrimary
}

func (e *Email) HasAddress(address EmailAddress) bool {
	return e != nil && e.Address == address
}

func (e *Email) HasID(id EmailID) bool {
	return e.IsActive() && e.ID == id
}

func (e *Email) ensureChangeAllowed(now time.Time) error {
	if e == nil {
		return errors.New("email is nil")
	}
	if !e.IsActive() {
		return errors.New("email is deleted")
	}
	if now.IsZero() || now.Before(e.UpdatedAt) {
		return errors.New("email change time is invalid")
	}

	return nil
}
