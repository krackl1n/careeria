package serviceaccount

import (
	"errors"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type ID uuid.UUID
type OrganizationID uuid.UUID
type CredentialID uuid.UUID

func (id ID) UUID() uuid.UUID             { return uuid.UUID(id) }
func (id OrganizationID) UUID() uuid.UUID { return uuid.UUID(id) }
func (id CredentialID) UUID() uuid.UUID   { return uuid.UUID(id) }

func (id ID) IsValid() bool             { return uuid.UUID(id) != uuid.Nil }
func (id OrganizationID) IsValid() bool { return uuid.UUID(id) != uuid.Nil }
func (id CredentialID) IsValid() bool   { return uuid.UUID(id) != uuid.Nil }

type AccountKey string

func NewAccountKey(value string) (AccountKey, error) {
	value, ok := domain.NormalizeToken(value)
	if !ok {
		return "", errors.New("service account key must be a non-empty token")
	}

	return AccountKey(value), nil
}

type Status string

const (
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
	StatusDeleted  Status = "deleted"
)

func ParseStatus(value string) (Status, error) {
	status := Status(value)
	switch status {
	case StatusActive, StatusDisabled, StatusDeleted:
		return status, nil
	default:
		return "", errors.New("unknown service account status")
	}
}

type CredentialType string

const CredentialTypeSharedSecret CredentialType = "shared_secret"

func ParseCredentialType(value string) (CredentialType, error) {
	credentialType := CredentialType(value)
	if credentialType != CredentialTypeSharedSecret {
		return "", errors.New("unknown service account credential type")
	}

	return credentialType, nil
}

type SecretHash string

func NewSecretHash(value string) (SecretHash, error) {
	if !domain.HasText(value) {
		return "", errors.New("service account secret hash is required")
	}

	return SecretHash(value), nil
}

type SecretPrefix string

func NewSecretPrefix(value string) (SecretPrefix, error) {
	value, ok := domain.NormalizeToken(value)
	if !ok {
		return "", errors.New("service account secret prefix must be a non-empty token")
	}

	return SecretPrefix(value), nil
}
