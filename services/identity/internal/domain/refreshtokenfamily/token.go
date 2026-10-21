package refreshtokenfamily

import (
	"errors"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

type RefreshToken struct {
	ID            TokenID
	ParentTokenID *TokenID
	Hash          TokenHash

	CreatedFromIP    *sessiondomain.IPAddress
	CreatedUserAgent *string

	IssuedAt  time.Time
	ExpiresAt time.Time
	RotatedAt *time.Time

	RevokedAt       *time.Time
	RevokeReason    *string
	ReuseDetectedAt *time.Time
}

var errTokenAlreadyRotated = errors.New("refresh token was already rotated")

func newRefreshToken(
	id TokenID,
	parentTokenID *TokenID,
	hash TokenHash,
	expiresAt time.Time,
	createdFromIP *sessiondomain.IPAddress,
	createdUserAgent *string,
	now time.Time,
) (RefreshToken, error) {
	if !id.IsValid() {
		return RefreshToken{}, errors.New("refresh token identifier is invalid")
	}
	if parentTokenID != nil && (!parentTokenID.IsValid() || *parentTokenID == id) {
		return RefreshToken{}, errors.New("refresh token parent identifier is invalid")
	}
	if now.IsZero() || !expiresAt.After(now) {
		return RefreshToken{}, errors.New("refresh token expiration is invalid")
	}

	return RefreshToken{
		ID:               id,
		ParentTokenID:    domain.ClonePointer(parentTokenID),
		Hash:             hash,
		IssuedAt:         now,
		ExpiresAt:        expiresAt,
		CreatedFromIP:    domain.ClonePointer(createdFromIP),
		CreatedUserAgent: domain.NormalizeOptionalText(createdUserAgent),
	}, nil
}

func (t *RefreshToken) ValidateForRotation(now time.Time) error {
	if t == nil {
		return errors.New("refresh token is nil")
	}
	if now.IsZero() || now.Before(t.latestChangedAt()) {
		return errors.New("refresh token use time is invalid")
	}
	if t.RotatedAt != nil {
		return errTokenAlreadyRotated
	}
	if t.RevokedAt != nil {
		return ErrTokenRevoked
	}
	if !now.Before(t.ExpiresAt) {
		return ErrTokenExpired
	}

	return nil
}

func (t *RefreshToken) MarkRotated(now time.Time) error {
	if err := t.ValidateForRotation(now); err != nil {
		return err
	}

	t.RotatedAt = domain.ClonePointer(&now)
	return nil
}

func (t *RefreshToken) RecordReuseDetected(now time.Time) error {
	if t == nil {
		return errors.New("refresh token is nil")
	}
	if t.RotatedAt == nil {
		return errors.New("refresh token was not rotated")
	}
	if now.IsZero() || now.Before(t.latestChangedAt()) {
		return errors.New("refresh token reuse time is invalid")
	}

	t.ReuseDetectedAt = domain.ClonePointer(&now)
	return nil
}

func (t *RefreshToken) Revoke(reason string, now time.Time) error {
	if t == nil {
		return errors.New("refresh token is nil")
	}
	if t.RevokedAt != nil {
		return nil
	}
	if now.IsZero() || now.Before(t.latestChangedAt()) {
		return errors.New("refresh token revocation time is invalid")
	}
	normalizedReason, ok := domain.NormalizeRequiredText(reason)
	if !ok {
		return errors.New("refresh token revoke reason is required")
	}

	t.RevokedAt = domain.ClonePointer(&now)
	t.RevokeReason = &normalizedReason
	return nil
}

func (t *RefreshToken) latestChangedAt() time.Time {
	if t == nil {
		return time.Time{}
	}
	latest := t.IssuedAt
	for _, value := range []*time.Time{t.RotatedAt, t.RevokedAt, t.ReuseDetectedAt} {
		if value != nil && value.After(latest) {
			latest = *value
		}
	}
	return latest
}
