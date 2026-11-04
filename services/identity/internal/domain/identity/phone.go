package identity

import (
	"errors"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type Phone struct {
	ID     PhoneID
	Number PhoneNumber

	IsPrimary bool

	CreatedAt  time.Time
	UpdatedAt  time.Time
	VerifiedAt *time.Time
	DeletedAt  *time.Time
}

func newPhone(id PhoneID, number PhoneNumber, primary bool, now time.Time) (Phone, error) {
	if !id.IsValid() || now.IsZero() {
		return Phone{}, errors.New("identity phone constructor arguments are invalid")
	}

	return Phone{
		ID:        id,
		Number:    number,
		IsPrimary: primary,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (p *Phone) Verify(now time.Time) error {
	if err := p.ensureChangeAllowed(now); err != nil {
		return err
	}
	if p.VerifiedAt != nil {
		return errors.New("phone is already verified")
	}

	p.VerifiedAt = domain.ClonePointer(&now)
	p.UpdatedAt = now
	return nil
}

func (p *Phone) Promote(now time.Time) (bool, error) {
	if err := p.ensureChangeAllowed(now); err != nil {
		return false, err
	}
	if p.VerifiedAt == nil {
		return false, errors.New("phone is not verified")
	}
	if p.IsPrimary {
		return false, nil
	}

	p.IsPrimary = true
	p.UpdatedAt = now
	return true, nil
}

func (p *Phone) Demote(now time.Time) error {
	if p == nil || !p.IsActive() || !p.IsPrimary {
		return nil
	}
	if err := p.ensureChangeAllowed(now); err != nil {
		return err
	}

	p.IsPrimary = false
	p.UpdatedAt = now
	return nil
}

func (p *Phone) Remove(now time.Time) error {
	if err := p.ensureChangeAllowed(now); err != nil {
		return err
	}

	p.IsPrimary = false
	p.DeletedAt = domain.ClonePointer(&now)
	p.UpdatedAt = now
	return nil
}

func (p *Phone) IsActive() bool {
	return p != nil && p.DeletedAt == nil
}

func (p *Phone) IsUsable() bool {
	return p.IsActive() && p.VerifiedAt != nil
}

func (p *Phone) IsPrimaryPhone() bool {
	return p.IsActive() && p.IsPrimary
}

func (p *Phone) HasNumber(number PhoneNumber) bool {
	return p != nil && p.Number == number
}

func (p *Phone) HasID(id PhoneID) bool {
	return p.IsActive() && p.ID == id
}

func (p *Phone) ensureChangeAllowed(now time.Time) error {
	if p == nil {
		return errors.New("phone is nil")
	}
	if !p.IsActive() {
		return errors.New("phone is deleted")
	}
	if now.IsZero() || now.Before(p.UpdatedAt) {
		return errors.New("phone change time is invalid")
	}

	return nil
}
