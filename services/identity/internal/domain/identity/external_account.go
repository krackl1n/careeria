package identity

import (
	"errors"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type ExternalAccount struct {
	ID       ExternalAccountID
	Provider ExternalProvider
	Subject  ExternalSubject

	LinkedAt time.Time

	LastAuthenticatedAt *time.Time

	DisabledAt *time.Time
	DeletedAt  *time.Time
}

func newExternalAccount(
	id ExternalAccountID,
	provider ExternalProvider,
	subject ExternalSubject,
	now time.Time,
) (ExternalAccount, error) {
	if !id.IsValid() || now.IsZero() {
		return ExternalAccount{}, errors.New("external identity account constructor arguments are invalid")
	}

	return ExternalAccount{
		ID:       id,
		Provider: provider,
		Subject:  subject,
		LinkedAt: now,
	}, nil
}

func (a *ExternalAccount) RecordAuthentication(now time.Time) error {
	if err := a.ensureChangeAllowed(now); err != nil {
		return err
	}
	if !a.IsUsable() {
		return ErrExternalAccountDisabled
	}

	a.LastAuthenticatedAt = domain.ClonePointer(&now)
	return nil
}

func (a *ExternalAccount) Disable(now time.Time) error {
	if err := a.ensureChangeAllowed(now); err != nil {
		return err
	}
	if a.DisabledAt != nil {
		return ErrExternalAccountDisabled
	}

	a.DisabledAt = domain.ClonePointer(&now)
	return nil
}

func (a *ExternalAccount) Enable(now time.Time) error {
	if err := a.ensureChangeAllowed(now); err != nil {
		return err
	}
	if a.DisabledAt == nil {
		return errors.New("account is not disabled")
	}

	a.DisabledAt = nil
	return nil
}

func (a *ExternalAccount) Unlink(now time.Time) error {
	if err := a.ensureChangeAllowed(now); err != nil {
		return err
	}

	a.DeletedAt = domain.ClonePointer(&now)
	return nil
}

func (a *ExternalAccount) IsActive() bool {
	return a != nil && a.DeletedAt == nil
}

func (a *ExternalAccount) IsUsable() bool {
	return a.IsActive() && a.DisabledAt == nil
}

func (a *ExternalAccount) Matches(provider ExternalProvider, subject ExternalSubject) bool {
	return a != nil && a.Provider == provider && a.Subject == subject
}

func (a *ExternalAccount) HasID(id ExternalAccountID) bool {
	return a.IsActive() && a.ID == id
}

func (a *ExternalAccount) ensureChangeAllowed(now time.Time) error {
	if a == nil {
		return errors.New("external identity account is nil")
	}
	if !a.IsActive() {
		return errors.New("external identity account is deleted")
	}
	if now.IsZero() || now.Before(a.latestChangedAt()) {
		return errors.New("external identity account change time is invalid")
	}

	return nil
}

func (a *ExternalAccount) latestChangedAt() time.Time {
	if a == nil {
		return time.Time{}
	}
	latest := a.LinkedAt
	for _, value := range []*time.Time{a.LastAuthenticatedAt, a.DisabledAt, a.DeletedAt} {
		if value != nil && value.After(latest) {
			latest = *value
		}
	}
	return latest
}
