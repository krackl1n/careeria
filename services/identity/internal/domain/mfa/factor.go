package mfa

import (
	"errors"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type Factor struct {
	ID   FactorID
	Type FactorType
	Name *string

	IsPrimary bool

	TOTP     *TOTP
	WebAuthn *WebAuthnCredential

	CreatedAt   time.Time
	UpdatedAt   time.Time
	ConfirmedAt *time.Time
	LastUsedAt  *time.Time
	DisabledAt  *time.Time
}

func newFactor(id FactorID, factorType FactorType, name *string, now time.Time) (Factor, error) {
	if !id.IsValid() {
		return Factor{}, errors.New("factor identifier is required")
	}
	if now.IsZero() {
		return Factor{}, errors.New("factor creation time is invalid")
	}

	return Factor{
		ID:        id,
		Type:      factorType,
		Name:      domain.NormalizeOptionalText(name),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (f *Factor) Rename(name *string, now time.Time) (bool, error) {
	if err := f.ensureChangeAllowed(now); err != nil {
		return false, err
	}
	normalizedName := domain.NormalizeOptionalText(name)
	if domain.EqualOptionalText(f.Name, normalizedName) {
		return false, nil
	}

	f.Name = normalizedName
	f.UpdatedAt = now
	return true, nil
}

func (f *Factor) Confirm(primary bool, now time.Time) error {
	if err := f.ensureChangeAllowed(now); err != nil {
		return err
	}
	if !f.IsEnabled() {
		return ErrFactorDisabled
	}
	if f.IsConfirmed() {
		return nil
	}

	f.ConfirmedAt = domain.ClonePointer(&now)
	f.IsPrimary = primary
	f.UpdatedAt = now
	return nil
}

func (f *Factor) Promote(now time.Time) (bool, error) {
	if err := f.ensureChangeAllowed(now); err != nil {
		return false, err
	}
	if !f.IsEnabled() {
		return false, ErrFactorDisabled
	}
	if !f.IsConfirmed() {
		return false, ErrFactorNotConfirmed
	}
	if f.IsPrimary {
		return false, nil
	}

	f.IsPrimary = true
	f.UpdatedAt = now
	return true, nil
}

func (f *Factor) Demote(now time.Time) error {
	if f == nil || !f.IsPrimary {
		return nil
	}
	if err := f.ensureChangeAllowed(now); err != nil {
		return err
	}

	f.IsPrimary = false
	f.UpdatedAt = now
	return nil
}

func (f *Factor) Disable(now time.Time) error {
	if err := f.ensureChangeAllowed(now); err != nil {
		return err
	}
	if !f.IsEnabled() {
		return ErrFactorDisabled
	}

	f.IsPrimary = false
	f.DisabledAt = domain.ClonePointer(&now)
	f.UpdatedAt = now
	return nil
}

func (f *Factor) Enable(now time.Time) error {
	if err := f.ensureChangeAllowed(now); err != nil {
		return err
	}
	if f.IsEnabled() {
		return nil
	}

	f.DisabledAt = nil
	f.UpdatedAt = now
	return nil
}

func (f *Factor) IsEnabled() bool {
	return f != nil && f.DisabledAt == nil
}

func (f *Factor) IsConfirmed() bool {
	return f != nil && f.ConfirmedAt != nil
}

func (f *Factor) IsPrimaryFactor() bool {
	return f != nil && f.IsPrimary
}

func (f *Factor) IsTOTP() bool {
	return f != nil && f.Type == FactorTypeTOTP && f.TOTP != nil
}

func (f *Factor) HasID(id FactorID) bool {
	return f != nil && f.ID == id
}

func (f *Factor) ChangedAt() time.Time {
	if f == nil {
		return time.Time{}
	}
	return f.UpdatedAt
}

func (f *Factor) ensureChangeAllowed(now time.Time) error {
	if f == nil {
		return errors.New("MFA factor is nil")
	}
	if now.IsZero() || now.Before(f.UpdatedAt) {
		return errors.New("MFA factor change time is invalid")
	}

	return nil
}

func (f *Factor) RecordTOTPUse(timeStep int64, now time.Time) error {
	if err := f.ensureChangeAllowed(now); err != nil {
		return err
	}
	if !f.IsTOTP() {
		return errors.New("factor type is invalid")
	}
	if !f.IsEnabled() {
		return ErrFactorDisabled
	}
	if !f.IsConfirmed() {
		return ErrFactorNotConfirmed
	}
	if err := f.TOTP.RecordUse(timeStep, now); err != nil {
		return err
	}

	f.LastUsedAt = domain.ClonePointer(&now)
	f.UpdatedAt = now
	return nil
}

func (f *Factor) RecordWebAuthnUse(signCount int64, backupState bool, now time.Time) error {
	if err := f.ensureChangeAllowed(now); err != nil {
		return err
	}
	if !f.IsWebAuthn() {
		return errors.New("factor type is invalid")
	}
	if !f.IsEnabled() {
		return ErrFactorDisabled
	}
	if !f.IsConfirmed() {
		return ErrFactorNotConfirmed
	}
	if err := f.WebAuthn.RecordUse(signCount, backupState, now); err != nil {
		return err
	}

	f.LastUsedAt = domain.ClonePointer(&now)
	f.UpdatedAt = now
	return nil
}

func (f *Factor) IsWebAuthn() bool {
	return f != nil && f.Type == FactorTypeWebAuthn && f.WebAuthn != nil
}

func (f *Factor) HasCredentialID(id CredentialID) bool {
	return f.IsWebAuthn() && f.WebAuthn.HasID(id)
}
