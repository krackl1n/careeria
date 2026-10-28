package authorizationgrant

import (
	"errors"

	"github.com/google/uuid"
)

type ID uuid.UUID

func (id ID) UUID() uuid.UUID { return uuid.UUID(id) }

func (id ID) IsValid() bool { return uuid.UUID(id) != uuid.Nil }

type Source string

const (
	SourceUserConsent         Source = "user_consent"
	SourceTrustedClientPolicy Source = "trusted_client_policy"
)

func ParseSource(value string) (Source, error) {
	source := Source(value)
	switch source {
	case SourceUserConsent, SourceTrustedClientPolicy:
		return source, nil
	default:
		return "", errors.New("unknown authorization grant source")
	}
}
