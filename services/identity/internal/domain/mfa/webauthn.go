package mfa

import (
	"bytes"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type WebAuthnCredential struct {
	CredentialID CredentialID
	PublicKey    PublicKey

	AAGUID     *uuid.UUID
	Transports []Transport

	SignCount      int64
	BackupEligible bool
	BackupState    bool

	CreatedAt  time.Time
	LastUsedAt *time.Time
}

func newWebAuthnFactor(
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
) (Factor, error) {
	factor, err := newFactor(id, FactorTypeWebAuthn, name, now)
	if err != nil {
		return Factor{}, err
	}
	credential, err := newWebAuthnCredential(
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
		return Factor{}, err
	}
	factor.WebAuthn = &credential
	return factor, nil
}

func newWebAuthnCredential(
	credentialID CredentialID,
	publicKey PublicKey,
	signCount int64,
	aaguid *uuid.UUID,
	transports []Transport,
	backupEligible bool,
	backupState bool,
	now time.Time,
) (WebAuthnCredential, error) {
	if signCount < 0 {
		return WebAuthnCredential{}, errors.New("signature counter is invalid")
	}
	if aaguid != nil && *aaguid == uuid.Nil {
		return WebAuthnCredential{}, errors.New("AAGUID is invalid")
	}
	if domain.HasDuplicates(transports) {
		return WebAuthnCredential{}, errors.New("transport is duplicated")
	}
	if backupState && !backupEligible {
		return WebAuthnCredential{}, errors.New("backup state requires backup eligibility")
	}
	if now.IsZero() {
		return WebAuthnCredential{}, errors.New("creation time is invalid")
	}

	return WebAuthnCredential{
		CredentialID:   CredentialID(bytes.Clone(credentialID)),
		PublicKey:      PublicKey(bytes.Clone(publicKey)),
		SignCount:      signCount,
		AAGUID:         domain.ClonePointer(aaguid),
		Transports:     slices.Clone(transports),
		BackupEligible: backupEligible,
		BackupState:    backupState,
		CreatedAt:      now,
	}, nil
}

func (c *WebAuthnCredential) RecordUse(signCount int64, backupState bool, now time.Time) error {
	if c == nil {
		return errors.New("WebAuthn credential is nil")
	}
	if now.IsZero() || now.Before(c.latestChangedAt()) {
		return errors.New("WebAuthn credential use time is invalid")
	}
	if signCount < 0 {
		return errors.New("signature counter is invalid")
	}
	if c.SignCount > 0 && signCount <= c.SignCount {
		return ErrWebAuthnCounterRollback
	}
	if backupState && !c.BackupEligible {
		return errors.New("backup state is invalid")
	}

	if signCount > c.SignCount {
		c.SignCount = signCount
	}
	c.BackupState = backupState
	c.LastUsedAt = domain.ClonePointer(&now)
	return nil
}

func (c *WebAuthnCredential) HasID(id CredentialID) bool {
	return c != nil && bytes.Equal(c.CredentialID, id)
}

func (c *WebAuthnCredential) latestChangedAt() time.Time {
	if c == nil {
		return time.Time{}
	}
	if c.LastUsedAt != nil && c.LastUsedAt.After(c.CreatedAt) {
		return *c.LastUsedAt
	}
	return c.CreatedAt
}
