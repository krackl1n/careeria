package recoverycodeset

import (
	"errors"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type ID uuid.UUID
type CodeID uuid.UUID

func (id ID) UUID() uuid.UUID     { return uuid.UUID(id) }
func (id CodeID) UUID() uuid.UUID { return uuid.UUID(id) }

func (id ID) IsValid() bool     { return uuid.UUID(id) != uuid.Nil }
func (id CodeID) IsValid() bool { return uuid.UUID(id) != uuid.Nil }

type CodeHash string

func NewCodeHash(value string) (CodeHash, error) {
	if !domain.HasText(value) {
		return "", errors.New("recovery code hash is required")
	}

	return CodeHash(value), nil
}

type CodeDefinition struct {
	ID   CodeID
	Hash CodeHash
}

type UseOutcome string

const (
	UseOutcomeUsed      UseOutcome = "used"
	UseOutcomeExhausted UseOutcome = "exhausted"
)
