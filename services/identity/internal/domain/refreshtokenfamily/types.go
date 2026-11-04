package refreshtokenfamily

import (
	"errors"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type ID uuid.UUID
type TokenID uuid.UUID

func (id ID) UUID() uuid.UUID      { return uuid.UUID(id) }
func (id TokenID) UUID() uuid.UUID { return uuid.UUID(id) }

func (id ID) IsValid() bool      { return uuid.UUID(id) != uuid.Nil }
func (id TokenID) IsValid() bool { return uuid.UUID(id) != uuid.Nil }

type TokenHash string

func NewTokenHash(value string) (TokenHash, error) {
	if !domain.HasText(value) {
		return "", errors.New("refresh token hash is required")
	}

	return TokenHash(value), nil
}

type RotationOutcome string

const (
	RotationOutcomeRotated       RotationOutcome = "rotated"
	RotationOutcomeReuseDetected RotationOutcome = "reuse_detected"
)
