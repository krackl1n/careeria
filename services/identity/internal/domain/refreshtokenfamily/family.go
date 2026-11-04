package refreshtokenfamily

import (
	"errors"
	"fmt"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

const reuseDetectedRevokeReason = "refresh_token_reuse_detected"

type RefreshTokenFamily struct {
	ID        ID
	SessionID sessiondomain.ID

	Tokens []RefreshToken

	CreatedAt         time.Time
	LastRotatedAt     *time.Time
	AbsoluteExpiresAt time.Time

	ReuseDetectedAt *time.Time
	RevokedAt       *time.Time
	RevokeReason    *string
}

func New(
	id ID,
	sessionID sessiondomain.ID,
	initialTokenID TokenID,
	initialTokenHash TokenHash,
	initialTokenExpiresAt time.Time,
	absoluteExpiresAt time.Time,
	createdFromIP *sessiondomain.IPAddress,
	createdUserAgent *string,
	now time.Time,
) (*RefreshTokenFamily, error) {
	if !id.IsValid() {
		return nil, errors.New("create refresh token family: identifier is required")
	}
	if !sessionID.IsValid() {
		return nil, errors.New("create refresh token family: session identifier is invalid")
	}
	if now.IsZero() || !absoluteExpiresAt.After(now) {
		return nil, errors.New("create refresh token family: absolute expiration is invalid")
	}
	if initialTokenExpiresAt.After(absoluteExpiresAt) {
		return nil, errors.New("create refresh token family: initial token exceeds family expiration")
	}

	initialToken, err := newRefreshToken(
		initialTokenID,
		nil,
		initialTokenHash,
		initialTokenExpiresAt,
		createdFromIP,
		createdUserAgent,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("create initial refresh token: %w", err)
	}

	return &RefreshTokenFamily{
		ID:                id,
		SessionID:         sessionID,
		Tokens:            []RefreshToken{initialToken},
		CreatedAt:         now,
		AbsoluteExpiresAt: absoluteExpiresAt,
	}, nil
}

func (f *RefreshTokenFamily) Revoke(reason string, now time.Time) error {
	if f == nil {
		return errors.New("revoke refresh token family: family is nil")
	}
	if f.RevokedAt != nil {
		return nil
	}
	if now.IsZero() || now.Before(f.latestChangedAt()) {
		return errors.New("revoke refresh token family: time is invalid")
	}
	var ok bool
	reason, ok = domain.NormalizeRequiredText(reason)
	if !ok {
		return errors.New("revoke refresh token family: reason is required")
	}

	if err := f.revoke(reason, now); err != nil {
		return fmt.Errorf("revoke refresh token family: %w", err)
	}
	return nil
}

func (f *RefreshTokenFamily) Rotate(
	presentedTokenID TokenID,
	nextTokenID TokenID,
	nextTokenHash TokenHash,
	nextTokenExpiresAt time.Time,
	createdFromIP *sessiondomain.IPAddress,
	createdUserAgent *string,
	now time.Time,
) (RotationOutcome, error) {
	if err := f.ensureRotationAllowed(now); err != nil {
		return "", fmt.Errorf("rotate refresh token: %w", err)
	}
	presentedIndex := f.tokenIndex(presentedTokenID)
	if presentedIndex < 0 {
		return "", fmt.Errorf("rotate refresh token: %w", ErrTokenNotFound)
	}
	presented := &f.Tokens[presentedIndex]
	if err := presented.ValidateForRotation(now); errors.Is(err, errTokenAlreadyRotated) {
		if detectErr := presented.RecordReuseDetected(now); detectErr != nil {
			return "", fmt.Errorf("rotate refresh token: record token reuse: %w", detectErr)
		}
		f.ReuseDetectedAt = domain.ClonePointer(&now)
		if revokeErr := f.revoke(reuseDetectedRevokeReason, now); revokeErr != nil {
			return "", fmt.Errorf("rotate refresh token: revoke reused token family: %w", revokeErr)
		}
		return RotationOutcomeReuseDetected, nil
	} else if err != nil {
		return "", fmt.Errorf("rotate refresh token: %w", err)
	}
	if f.tokenIndex(nextTokenID) >= 0 {
		return "", errors.New("rotate refresh token: next token identifier already exists")
	}
	if f.hasTokenHash(nextTokenHash) {
		return "", errors.New("rotate refresh token: next token hash already exists")
	}
	if nextTokenExpiresAt.After(f.AbsoluteExpiresAt) {
		return "", errors.New("rotate refresh token: next token expiration exceeds family expiration")
	}

	nextToken, err := newRefreshToken(
		nextTokenID,
		&presentedTokenID,
		nextTokenHash,
		nextTokenExpiresAt,
		createdFromIP,
		createdUserAgent,
		now,
	)
	if err != nil {
		return "", fmt.Errorf("rotate refresh token: %w", err)
	}

	if err := presented.MarkRotated(now); err != nil {
		return "", fmt.Errorf("rotate refresh token: %w", err)
	}
	f.Tokens = append(f.Tokens, nextToken)
	f.LastRotatedAt = domain.ClonePointer(&now)
	return RotationOutcomeRotated, nil
}

func (f *RefreshTokenFamily) TokenIDByHash(hash TokenHash) (TokenID, error) {
	if f == nil {
		return TokenID{}, errors.New("find refresh token: family is nil")
	}
	for index := range f.Tokens {
		if f.Tokens[index].Hash == hash {
			return f.Tokens[index].ID, nil
		}
	}

	return TokenID{}, ErrTokenNotFound
}

func (f *RefreshTokenFamily) ensureRotationAllowed(now time.Time) error {
	if f == nil {
		return errors.New("validate refresh token family for rotation: family is nil")
	}
	if now.IsZero() || now.Before(f.latestChangedAt()) {
		return errors.New("validate refresh token family for rotation: time is invalid")
	}
	if f.RevokedAt != nil {
		return fmt.Errorf("validate refresh token family for rotation: %w", ErrRevoked)
	}
	if !now.Before(f.AbsoluteExpiresAt) {
		return fmt.Errorf("validate refresh token family for rotation: %w", ErrExpired)
	}

	return nil
}

func (f *RefreshTokenFamily) revoke(reason string, now time.Time) error {
	for index := range f.Tokens {
		if err := f.Tokens[index].Revoke(reason, now); err != nil {
			return err
		}
	}
	f.RevokedAt = domain.ClonePointer(&now)
	f.RevokeReason = &reason
	return nil
}

func (f *RefreshTokenFamily) latestChangedAt() time.Time {
	if f == nil {
		return time.Time{}
	}
	latest := f.CreatedAt
	for _, value := range []*time.Time{f.LastRotatedAt, f.ReuseDetectedAt, f.RevokedAt} {
		if value != nil && value.After(latest) {
			latest = *value
		}
	}
	return latest
}

func (f *RefreshTokenFamily) tokenIndex(id TokenID) int {
	for index := range f.Tokens {
		if f.Tokens[index].ID == id {
			return index
		}
	}
	return -1
}

func (f *RefreshTokenFamily) hasTokenHash(hash TokenHash) bool {
	for index := range f.Tokens {
		if f.Tokens[index].Hash == hash {
			return true
		}
	}
	return false
}
