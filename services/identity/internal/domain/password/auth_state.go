package password

import (
	"errors"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type AuthState struct {
	FailedAttempts int
	LastFailedAt   *time.Time

	LockedUntil *time.Time
	LockReason  *LockReason

	UpdatedAt time.Time
}

func newAuthState(now time.Time) (AuthState, error) {
	if now.IsZero() {
		return AuthState{}, errors.New("time is invalid")
	}
	return AuthState{UpdatedAt: now}, nil
}

func (s *AuthState) ValidateForAuthentication(now time.Time) error {
	if s == nil {
		return errors.New("password authentication state is nil")
	}
	if now.IsZero() || now.Before(s.UpdatedAt) {
		return errors.New("password authentication state use time is invalid")
	}
	if s.LockedUntil != nil && now.Before(*s.LockedUntil) {
		return ErrLocked
	}

	return nil
}

func (s *AuthState) RecordFailedAttempt(policy FailurePolicy, now time.Time) (bool, error) {
	if s == nil {
		return false, errors.New("password authentication state is nil")
	}
	if err := policy.Validate(); err != nil {
		return false, err
	}
	if err := s.ValidateForAuthentication(now); err != nil {
		return false, err
	}
	if s.LockedUntil != nil {
		s.Reset(now)
	}

	s.FailedAttempts++
	s.LastFailedAt = domain.ClonePointer(&now)
	s.UpdatedAt = now

	locked := s.FailedAttempts >= policy.MaxAttempts
	if locked {
		lockedUntil := now.Add(policy.LockDuration)
		reason := LockReasonFailedAttempts
		s.LockedUntil = domain.ClonePointer(&lockedUntil)
		s.LockReason = domain.ClonePointer(&reason)
	}
	return locked, nil
}

func (s *AuthState) Unlock(now time.Time) error {
	if s == nil {
		return errors.New("password authentication state is nil")
	}
	if now.IsZero() || now.Before(s.UpdatedAt) {
		return errors.New("password authentication state change time is invalid")
	}
	if s.LockedUntil == nil {
		return errors.New("password is not locked")
	}

	s.Reset(now)
	return nil
}

func (s *AuthState) Reset(now time.Time) {
	*s = AuthState{UpdatedAt: now}
}

func (s *AuthState) ChangedAt() time.Time {
	if s == nil {
		return time.Time{}
	}
	return s.UpdatedAt
}
