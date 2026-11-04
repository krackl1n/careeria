package mfa

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

type MFA struct {
	IdentityID identitydomain.ID

	Factors []Factor
}

func New(identityID identitydomain.ID) (*MFA, error) {
	if !identityID.IsValid() {
		return nil, errors.New("create MFA configuration: identity identifier is invalid")
	}

	return &MFA{IdentityID: identityID}, nil
}

func (m *MFA) RenameFactor(id FactorID, name *string, now time.Time) error {
	if err := m.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("rename MFA factor: %w", err)
	}
	index := m.factorIndex(id)
	if index < 0 {
		return fmt.Errorf("rename MFA factor: %w", ErrFactorNotFound)
	}
	changed, err := m.Factors[index].Rename(name, now)
	if err != nil {
		return fmt.Errorf("rename MFA factor: %w", err)
	}
	if !changed {
		return nil
	}

	return nil
}

func (m *MFA) ConfirmFactor(id FactorID, now time.Time) error {
	if err := m.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("confirm MFA factor: %w", err)
	}
	index := m.factorIndex(id)
	if index < 0 {
		return fmt.Errorf("confirm MFA factor: %w", ErrFactorNotFound)
	}
	if err := m.Factors[index].Confirm(m.primaryFactorIndex() < 0, now); err != nil {
		return fmt.Errorf("confirm MFA factor: %w", err)
	}
	return nil
}

func (m *MFA) SetPrimaryFactor(id FactorID, now time.Time) error {
	if err := m.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("set primary MFA factor: %w", err)
	}
	index := m.factorIndex(id)
	if index < 0 {
		return fmt.Errorf("set primary MFA factor: %w", ErrFactorNotFound)
	}
	changed, err := m.Factors[index].Promote(now)
	if err != nil {
		return fmt.Errorf("set primary MFA factor: %w", err)
	}
	if !changed {
		return nil
	}

	for factorIndex := range m.Factors {
		if factorIndex != index {
			if err := m.Factors[factorIndex].Demote(now); err != nil {
				return fmt.Errorf("set primary MFA factor: %w", err)
			}
		}
	}
	return nil
}

func (m *MFA) DisableFactor(id FactorID, now time.Time) error {
	if err := m.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("disable MFA factor: %w", err)
	}
	index := m.factorIndex(id)
	if index < 0 {
		return fmt.Errorf("disable MFA factor: %w", ErrFactorNotFound)
	}
	if err := m.Factors[index].Disable(now); err != nil {
		return fmt.Errorf("disable MFA factor: %w", err)
	}

	return nil
}

func (m *MFA) EnableFactor(id FactorID, now time.Time) error {
	if err := m.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("enable MFA factor: %w", err)
	}
	index := m.factorIndex(id)
	if index < 0 {
		return fmt.Errorf("enable MFA factor: %w", ErrFactorNotFound)
	}
	if m.Factors[index].IsEnabled() {
		return nil
	}
	if m.Factors[index].IsTOTP() && m.enabledTOTPIndex() >= 0 {
		return fmt.Errorf("enable MFA factor: %w", ErrEnabledTOTPAlreadyExists)
	}

	if err := m.Factors[index].Enable(now); err != nil {
		return fmt.Errorf("enable MFA factor: %w", err)
	}
	return nil
}

func (m *MFA) EnrollTOTP(
	id FactorID,
	name *string,
	secretEncrypted EncryptedSecret,
	encryptionKeyRef EncryptionKeyRef,
	algorithm TOTPAlgorithm,
	digits int,
	periodSeconds int,
	now time.Time,
) error {
	if err := m.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("enroll TOTP factor: %w", err)
	}
	factor, err := newTOTPFactor(
		id,
		name,
		secretEncrypted,
		encryptionKeyRef,
		algorithm,
		digits,
		periodSeconds,
		now,
	)
	if err != nil {
		return fmt.Errorf("enroll TOTP factor: %w", err)
	}
	if m.factorIndex(factor.ID) >= 0 {
		return fmt.Errorf("enroll TOTP factor: %w", ErrFactorAlreadyExists)
	}
	if m.enabledTOTPIndex() >= 0 {
		return fmt.Errorf("enroll TOTP factor: %w", ErrEnabledTOTPAlreadyExists)
	}
	m.Factors = append(m.Factors, factor)
	return nil
}

func (m *MFA) RecordTOTPUse(id FactorID, timeStep int64, now time.Time) error {
	if err := m.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("record TOTP factor use: %w", err)
	}
	index := m.factorIndex(id)
	if index < 0 {
		return fmt.Errorf("record TOTP factor use: %w", ErrFactorNotFound)
	}
	if err := m.Factors[index].RecordTOTPUse(timeStep, now); err != nil {
		return fmt.Errorf("record TOTP factor use: %w", err)
	}

	return nil
}

func (m *MFA) EnrollWebAuthn(
	id FactorID,
	name *string,
	credentialID CredentialID,
	publicKey PublicKey,
	signCount int64,
	aaguid *uuid.UUID,
	transports []Transport,
	backupEligible bool,
	backupState bool,
	now time.Time,
) error {
	if err := m.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("enroll WebAuthn credential: %w", err)
	}
	factor, err := newWebAuthnFactor(
		id,
		name,
		credentialID,
		publicKey,
		signCount,
		aaguid,
		transports,
		backupEligible,
		backupState,
		now,
	)
	if err != nil {
		return fmt.Errorf("enroll WebAuthn credential: %w", err)
	}
	if m.factorIndex(factor.ID) >= 0 {
		return fmt.Errorf("enroll WebAuthn credential: %w", ErrFactorAlreadyExists)
	}
	if m.hasCredentialID(credentialID) {
		return fmt.Errorf("enroll WebAuthn credential: %w", ErrCredentialAlreadyExists)
	}
	m.Factors = append(m.Factors, factor)
	return nil
}

func (m *MFA) RecordWebAuthnUse(
	id FactorID,
	signCount int64,
	backupState bool,
	now time.Time,
) error {
	if err := m.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("record WebAuthn credential use: %w", err)
	}
	index := m.factorIndex(id)
	if index < 0 {
		return fmt.Errorf("record WebAuthn credential use: %w", ErrFactorNotFound)
	}
	if err := m.Factors[index].RecordWebAuthnUse(signCount, backupState, now); err != nil {
		return fmt.Errorf("record WebAuthn credential use: %w", err)
	}

	return nil
}

func (m *MFA) ensureChangeAllowed(now time.Time) error {
	if m == nil {
		return errors.New("MFA configuration is nil")
	}
	if now.IsZero() || now.Before(m.latestUpdatedAt()) {
		return errors.New("MFA change time is invalid")
	}

	return nil
}

func (m *MFA) latestUpdatedAt() time.Time {
	if m == nil {
		return time.Time{}
	}
	latest := time.Time{}
	for index := range m.Factors {
		if changedAt := m.Factors[index].ChangedAt(); changedAt.After(latest) {
			latest = changedAt
		}
	}
	return latest
}

func (m *MFA) factorIndex(id FactorID) int {
	for index := range m.Factors {
		if m.Factors[index].HasID(id) {
			return index
		}
	}
	return -1
}

func (m *MFA) primaryFactorIndex() int {
	for index := range m.Factors {
		if m.Factors[index].IsPrimaryFactor() {
			return index
		}
	}
	return -1
}

func (m *MFA) enabledTOTPIndex() int {
	for index := range m.Factors {
		if m.Factors[index].IsTOTP() && m.Factors[index].IsEnabled() {
			return index
		}
	}
	return -1
}

func (m *MFA) hasCredentialID(id CredentialID) bool {
	for index := range m.Factors {
		if m.Factors[index].HasCredentialID(id) {
			return true
		}
	}
	return false
}
