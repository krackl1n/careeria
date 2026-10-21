package signingkey

import (
	"errors"
	"fmt"
	"slices"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type SigningKey struct {
	ID ID

	KeyID         KeyID
	Algorithm     Algorithm
	PublicJWK     PublicJWK
	PrivateKeyRef PrivateKeyRef

	Status Status

	CreatedAt time.Time

	NotBefore time.Time
	ExpiresAt *time.Time

	ActivatedAt   *time.Time
	RetiringAt    *time.Time
	RetiredAt     *time.Time
	CompromisedAt *time.Time
}

func New(
	id ID,
	keyID KeyID,
	algorithm Algorithm,
	publicJWK PublicJWK,
	privateKeyRef PrivateKeyRef,
	notBefore time.Time,
	expiresAt *time.Time,
	now time.Time,
) (*SigningKey, error) {
	if !id.IsValid() {
		return nil, errors.New("create signing key: identifier is required")
	}
	if now.IsZero() || notBefore.Before(now) {
		return nil, errors.New("create signing key: validity start is invalid")
	}
	if expiresAt != nil && !expiresAt.After(notBefore) {
		return nil, errors.New("create signing key: expiration is invalid")
	}
	if err := validatePublicJWKMetadata(publicJWK, keyID, algorithm); err != nil {
		return nil, fmt.Errorf("create signing key: validate public JWK metadata: %w", err)
	}

	return &SigningKey{
		ID: id,

		KeyID:         keyID,
		Algorithm:     algorithm,
		PublicJWK:     slices.Clone(publicJWK),
		PrivateKeyRef: privateKeyRef,

		Status: StatusPending,

		CreatedAt: now,
		NotBefore: notBefore,
		ExpiresAt: domain.ClonePointer(expiresAt),
	}, nil
}

func (k *SigningKey) Activate(now time.Time) error {
	if err := k.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("activate signing key: %w", err)
	}
	if k.Status != StatusPending {
		return fmt.Errorf("activate signing key: %w", ErrInvalidStatus)
	}
	if now.Before(k.NotBefore) {
		return fmt.Errorf("activate signing key: %w", ErrNotYetValid)
	}
	if k.ExpiresAt != nil && !now.Before(*k.ExpiresAt) {
		return fmt.Errorf("activate signing key: %w", ErrExpired)
	}

	k.Status = StatusActive
	k.ActivatedAt = domain.ClonePointer(&now)
	return nil
}

func (k *SigningKey) StartRetirement(now time.Time) error {
	if err := k.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("start signing key retirement: %w", err)
	}
	if k.Status != StatusActive {
		return fmt.Errorf("start signing key retirement: %w", ErrInvalidStatus)
	}

	k.Status = StatusRetiring
	k.RetiringAt = domain.ClonePointer(&now)
	return nil
}

func (k *SigningKey) Retire(now time.Time) error {
	if err := k.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("retire signing key: %w", err)
	}
	if k.Status != StatusRetiring {
		return fmt.Errorf("retire signing key: %w", ErrInvalidStatus)
	}

	k.Status = StatusRetired
	k.RetiredAt = domain.ClonePointer(&now)
	return nil
}

func (k *SigningKey) MarkCompromised(now time.Time) error {
	if k == nil {
		return errors.New("mark signing key compromised: signing key is nil")
	}
	if k.Status == StatusCompromised {
		return nil
	}
	if err := k.ensureChangeAllowed(now); err != nil {
		return fmt.Errorf("mark signing key compromised: %w", err)
	}

	k.Status = StatusCompromised
	k.CompromisedAt = domain.ClonePointer(&now)
	return nil
}

func (k *SigningKey) ValidateForSigning(now time.Time) error {
	if k == nil {
		return errors.New("validate signing key for signing: signing key is nil")
	}
	if now.IsZero() || now.Before(k.latestChangedAt()) {
		return errors.New("validate signing key for signing: time is invalid")
	}
	if k.Status == StatusCompromised {
		return fmt.Errorf("validate signing key for signing: %w", ErrCompromised)
	}
	if k.Status != StatusActive {
		return fmt.Errorf("validate signing key for signing: %w", ErrInvalidStatus)
	}
	if now.Before(k.NotBefore) {
		return fmt.Errorf("validate signing key for signing: %w", ErrNotYetValid)
	}
	if k.ExpiresAt != nil && !now.Before(*k.ExpiresAt) {
		return fmt.Errorf("validate signing key for signing: %w", ErrExpired)
	}

	return nil
}

func (k *SigningKey) ValidateForVerification(now time.Time) error {
	if k == nil {
		return errors.New("validate signing key for verification: signing key is nil")
	}
	if now.IsZero() || now.Before(k.latestChangedAt()) {
		return errors.New("validate signing key for verification: time is invalid")
	}
	if k.Status == StatusCompromised {
		return fmt.Errorf("validate signing key for verification: %w", ErrCompromised)
	}
	if k.Status != StatusActive && k.Status != StatusRetiring {
		return fmt.Errorf("validate signing key for verification: %w", ErrInvalidStatus)
	}
	if now.Before(k.NotBefore) {
		return fmt.Errorf("validate signing key for verification: %w", ErrNotYetValid)
	}
	if k.ExpiresAt != nil && !now.Before(*k.ExpiresAt) {
		return fmt.Errorf("validate signing key for verification: %w", ErrExpired)
	}

	return nil
}

func (k *SigningKey) ensureChangeAllowed(now time.Time) error {
	if k == nil {
		return errors.New("validate signing key for change: signing key is nil")
	}
	if now.IsZero() || now.Before(k.latestChangedAt()) {
		return errors.New("validate signing key for change: time is invalid")
	}

	return nil
}

func (k *SigningKey) latestChangedAt() time.Time {
	if k == nil {
		return time.Time{}
	}
	latest := k.latestLifecycleAtExcludingCompromise()
	if k.CompromisedAt != nil && k.CompromisedAt.After(latest) {
		latest = *k.CompromisedAt
	}
	return latest
}

func (k *SigningKey) latestLifecycleAtExcludingCompromise() time.Time {
	latest := k.CreatedAt
	for _, value := range []*time.Time{k.ActivatedAt, k.RetiringAt, k.RetiredAt} {
		if value != nil && value.After(latest) {
			latest = *value
		}
	}
	return latest
}
