package phoneverificationchallenge

import (
	"errors"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type ID uuid.UUID

func (id ID) UUID() uuid.UUID { return uuid.UUID(id) }

func (id ID) IsValid() bool { return uuid.UUID(id) != uuid.Nil }

type TokenHash string

func NewTokenHash(value string) (TokenHash, error) {
	if !domain.HasText(value) {
		return "", errors.New("phone verification token hash is required")
	}
	return TokenHash(value), nil
}

type AttemptOutcome string

const (
	AttemptOutcomeRecorded  AttemptOutcome = "recorded"
	AttemptOutcomeExhausted AttemptOutcome = "exhausted"
)
