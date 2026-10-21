package password

import (
	"errors"
	"fmt"
	"time"

	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

type Password struct {
	IdentityID identitydomain.ID

	Credential Credential
	AuthState  AuthState
}

func New(
	identityID identitydomain.ID,
	hash Hash,
	mustChange bool,
	expiresAt *time.Time,
	now time.Time,
) (*Password, error) {
	if !identityID.IsValid() {
		return nil, errors.New("create password: identity identifier is invalid")
	}
	credential, err := newCredential(hash, mustChange, expiresAt, now)
	if err != nil {
		return nil, fmt.Errorf("create password: %w", err)
	}
	authState, err := newAuthState(now)
	if err != nil {
		return nil, fmt.Errorf("create password: %w", err)
	}

	return &Password{
		IdentityID: identityID,
		Credential: credential,
		AuthState:  authState,
	}, nil
}

func (p *Password) Change(hash Hash, mustChange bool, expiresAt *time.Time, now time.Time) error {
	if err := p.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("change password: %w", err)
	}
	if err := p.Credential.Change(hash, mustChange, expiresAt, now); err != nil {
		return fmt.Errorf("change password: %w", err)
	}

	p.AuthState.Reset(now)
	return nil
}

func (p *Password) RequireChange(now time.Time) error {
	if err := p.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("require password change: %w", err)
	}
	changed, err := p.Credential.RequireChange(now)
	if err != nil {
		return fmt.Errorf("require password change: %w", err)
	}
	if !changed {
		return nil
	}

	return nil
}

func (p *Password) MarkCompromised(now time.Time) error {
	if err := p.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("mark password compromised: %w", err)
	}
	changed, err := p.Credential.MarkCompromised(now)
	if err != nil {
		return fmt.Errorf("mark password compromised: %w", err)
	}
	if !changed {
		return nil
	}

	return nil
}

func (p *Password) RecordFailedAttempt(policy FailurePolicy, now time.Time) (bool, error) {
	if err := p.ensureChangeAllowed(now); err != nil {
		return false, fmt.Errorf("record failed password attempt: %w", err)
	}
	if err := policy.Validate(); err != nil {
		return false, fmt.Errorf("record failed password attempt: %w", err)
	}
	if err := p.ensureAuthenticationAllowed(now); err != nil {
		return false, fmt.Errorf("record failed password attempt: %w", err)
	}

	locked, err := p.AuthState.RecordFailedAttempt(policy, now)
	if err != nil {
		return false, fmt.Errorf("record failed password attempt: %w", err)
	}
	return locked, nil
}

func (p *Password) RecordSuccessfulAuthentication(now time.Time) error {
	if err := p.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("record successful password authentication: %w", err)
	}
	if err := p.ensureAuthenticationAllowed(now); err != nil {
		return fmt.Errorf("record successful password authentication: %w", err)
	}

	p.AuthState.Reset(now)
	return nil
}

func (p *Password) Unlock(now time.Time) error {
	if err := p.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("unlock password authentication: %w", err)
	}
	if err := p.AuthState.Unlock(now); err != nil {
		return fmt.Errorf("unlock password authentication: %w", err)
	}

	return nil
}

func (p *Password) ensureAuthenticationAllowed(now time.Time) error {
	if p == nil {
		return errors.New("validate password for authentication: password is nil")
	}
	if now.IsZero() || now.Before(p.latestUpdatedAt()) {
		return errors.New("validate password for authentication: time is invalid")
	}
	if err := p.AuthState.ValidateForAuthentication(now); err != nil {
		return fmt.Errorf("validate password for authentication: %w", err)
	}
	if err := p.Credential.ValidateForAuthentication(now); err != nil {
		return fmt.Errorf("validate password for authentication: %w", err)
	}

	return nil
}

func (p *Password) ensureChangeAllowed(now time.Time) error {
	if p == nil {
		return errors.New("password is nil")
	}
	if now.IsZero() || now.Before(p.latestUpdatedAt()) {
		return errors.New("password change time is invalid")
	}

	return nil
}

func (p *Password) latestUpdatedAt() time.Time {
	if p == nil {
		return time.Time{}
	}
	if p.AuthState.ChangedAt().After(p.Credential.ChangedAt()) {
		return p.AuthState.ChangedAt()
	}
	return p.Credential.ChangedAt()
}
