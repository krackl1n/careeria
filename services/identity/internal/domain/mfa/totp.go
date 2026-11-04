package mfa

import (
	"bytes"
	"errors"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type TOTP struct {
	SecretEncrypted  EncryptedSecret
	EncryptionKeyRef EncryptionKeyRef

	Algorithm     TOTPAlgorithm
	Digits        int
	PeriodSeconds int

	LastUsedTimeStep *int64
	UpdatedAt        time.Time
}

func newTOTPFactor(
	id FactorID,
	name *string,
	secretEncrypted EncryptedSecret,
	encryptionKeyRef EncryptionKeyRef,
	algorithm TOTPAlgorithm,
	digits int,
	periodSeconds int,
	now time.Time,
) (Factor, error) {
	factor, err := newFactor(id, FactorTypeTOTP, name, now)
	if err != nil {
		return Factor{}, err
	}
	totp, err := newTOTP(secretEncrypted, encryptionKeyRef, algorithm, digits, periodSeconds, now)
	if err != nil {
		return Factor{}, err
	}
	factor.TOTP = &totp
	return factor, nil
}

func newTOTP(
	secretEncrypted EncryptedSecret,
	encryptionKeyRef EncryptionKeyRef,
	algorithm TOTPAlgorithm,
	digits int,
	periodSeconds int,
	now time.Time,
) (TOTP, error) {
	if digits != 6 && digits != 8 {
		return TOTP{}, errors.New("digits must be 6 or 8")
	}
	if periodSeconds <= 0 {
		return TOTP{}, errors.New("period must be positive")
	}
	if now.IsZero() {
		return TOTP{}, errors.New("creation time is invalid")
	}

	return TOTP{
		SecretEncrypted:  EncryptedSecret(bytes.Clone(secretEncrypted)),
		EncryptionKeyRef: encryptionKeyRef,
		Algorithm:        algorithm,
		Digits:           digits,
		PeriodSeconds:    periodSeconds,
		UpdatedAt:        now,
	}, nil
}

func (t *TOTP) RecordUse(timeStep int64, now time.Time) error {
	if t == nil {
		return errors.New("TOTP configuration is nil")
	}
	if now.IsZero() || now.Before(t.UpdatedAt) {
		return errors.New("TOTP use time is invalid")
	}
	if timeStep < 0 {
		return errors.New("time step is invalid")
	}
	if t.LastUsedTimeStep != nil && timeStep <= *t.LastUsedTimeStep {
		return ErrTOTPReplay
	}

	t.LastUsedTimeStep = domain.ClonePointer(&timeStep)
	t.UpdatedAt = now
	return nil
}
