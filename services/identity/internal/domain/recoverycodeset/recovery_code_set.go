package recoverycodeset

import (
	"errors"
	"fmt"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

type RecoveryCodeSet struct {
	ID         ID
	IdentityID identitydomain.ID

	Codes []RecoveryCode

	CreatedAt  time.Time
	ExpiresAt  *time.Time
	DisabledAt *time.Time
}

func New(
	id ID,
	identityID identitydomain.ID,
	definitions []CodeDefinition,
	expiresAt *time.Time,
	now time.Time,
) (*RecoveryCodeSet, error) {
	if !id.IsValid() {
		return nil, errors.New("create recovery code set: identifier is required")
	}
	if !identityID.IsValid() {
		return nil, errors.New("create recovery code set: identity identifier is invalid")
	}
	if now.IsZero() {
		return nil, errors.New("create recovery code set: time is invalid")
	}
	if expiresAt != nil && !expiresAt.After(now) {
		return nil, errors.New("create recovery code set: expiration must follow creation")
	}
	if len(definitions) == 0 {
		return nil, errors.New("create recovery code set: at least one code is required")
	}
	ids := make(map[CodeID]struct{}, len(definitions))
	hashes := make(map[CodeHash]struct{}, len(definitions))
	codes := make([]RecoveryCode, 0, len(definitions))
	for _, definition := range definitions {
		code, err := newRecoveryCode(definition.ID, definition.Hash, now)
		if err != nil {
			return nil, fmt.Errorf("create recovery code set: %w", err)
		}
		if _, exists := ids[code.ID]; exists {
			return nil, errors.New("create recovery code set: code identifier is duplicated")
		}
		if _, exists := hashes[code.Hash]; exists {
			return nil, errors.New("create recovery code set: code hash is duplicated")
		}
		ids[code.ID] = struct{}{}
		hashes[code.Hash] = struct{}{}
		codes = append(codes, code)
	}

	entity := &RecoveryCodeSet{
		ID:         id,
		IdentityID: identityID,
		Codes:      codes,
		CreatedAt:  now,
		ExpiresAt:  domain.ClonePointer(expiresAt),
	}
	return entity, nil
}

func (s *RecoveryCodeSet) Disable(now time.Time) error {
	if s == nil {
		return errors.New("disable recovery code set: recovery code set is nil")
	}
	if s.DisabledAt != nil {
		return nil
	}
	if now.IsZero() || now.Before(s.latestChangedAt()) {
		return errors.New("disable recovery code set: time is invalid")
	}

	s.DisabledAt = domain.ClonePointer(&now)
	return nil
}

func (s *RecoveryCodeSet) UseCode(id CodeID, now time.Time) (UseOutcome, error) {
	if s == nil {
		return "", errors.New("use recovery code: recovery code set is nil")
	}
	if now.IsZero() || now.Before(s.latestChangedAt()) {
		return "", errors.New("use recovery code: time is invalid")
	}
	if s.DisabledAt != nil {
		return "", fmt.Errorf("use recovery code: %w", ErrDisabled)
	}
	if s.ExpiresAt != nil && !now.Before(*s.ExpiresAt) {
		return "", fmt.Errorf("use recovery code: %w", ErrExpired)
	}
	index := s.codeIndex(id)
	if index < 0 {
		return "", fmt.Errorf("use recovery code: %w", ErrCodeNotFound)
	}
	if err := s.Codes[index].Use(now); err != nil {
		return "", fmt.Errorf("use recovery code: %w", err)
	}

	outcome := UseOutcomeUsed
	if s.unusedCodeCount() == 0 {
		s.DisabledAt = domain.ClonePointer(&now)
		outcome = UseOutcomeExhausted
	}
	return outcome, nil
}

func (s *RecoveryCodeSet) latestChangedAt() time.Time {
	if s == nil {
		return time.Time{}
	}
	latest := s.latestCodeUseAt()
	if s.DisabledAt != nil && s.DisabledAt.After(latest) {
		latest = *s.DisabledAt
	}
	return latest
}

func (s *RecoveryCodeSet) latestCodeUseAt() time.Time {
	if s == nil {
		return time.Time{}
	}
	latest := s.CreatedAt
	for index := range s.Codes {
		if changedAt := s.Codes[index].ChangedAt(); changedAt.After(latest) {
			latest = changedAt
		}
	}
	return latest
}

func (s *RecoveryCodeSet) codeIndex(id CodeID) int {
	for index := range s.Codes {
		if s.Codes[index].ID == id {
			return index
		}
	}
	return -1
}

func (s *RecoveryCodeSet) unusedCodeCount() int {
	count := 0
	for index := range s.Codes {
		if !s.Codes[index].IsUsed() {
			count++
		}
	}
	return count
}
