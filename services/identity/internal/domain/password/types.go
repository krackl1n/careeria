package password

import (
	"errors"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type Hash string

func NewHash(value string) (Hash, error) {
	if !domain.HasText(value) {
		return "", errors.New("password hash is required")
	}

	return Hash(value), nil
}

type LockReason string

const LockReasonFailedAttempts LockReason = "failed_attempts"

func NewLockReason(value string) (LockReason, error) {
	value, ok := domain.NormalizeRequiredText(value)
	if !ok {
		return "", errors.New("password lock reason is required")
	}

	return LockReason(value), nil
}

type FailurePolicy struct {
	MaxAttempts  int
	LockDuration time.Duration
}

func (p FailurePolicy) Validate() error {
	if p.MaxAttempts <= 0 {
		return errors.New("max attempts must be positive")
	}
	if p.LockDuration <= 0 {
		return errors.New("lock duration must be positive")
	}
	return nil
}
