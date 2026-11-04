package mfa

import (
	"bytes"
	"errors"
	"strings"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

type FactorID uuid.UUID

func (id FactorID) UUID() uuid.UUID { return uuid.UUID(id) }

func (id FactorID) IsValid() bool { return uuid.UUID(id) != uuid.Nil }

type FactorType string

const (
	FactorTypeTOTP     FactorType = "totp"
	FactorTypeWebAuthn FactorType = "webauthn"
)

func ParseFactorType(value string) (FactorType, error) {
	factorType := FactorType(value)
	switch factorType {
	case FactorTypeTOTP, FactorTypeWebAuthn:
		return factorType, nil
	default:
		return "", errors.New("unknown MFA factor type")
	}
}

type TOTPAlgorithm string

const (
	TOTPAlgorithmSHA1   TOTPAlgorithm = "sha1"
	TOTPAlgorithmSHA256 TOTPAlgorithm = "sha256"
	TOTPAlgorithmSHA512 TOTPAlgorithm = "sha512"
)

func ParseTOTPAlgorithm(value string) (TOTPAlgorithm, error) {
	algorithm := TOTPAlgorithm(value)
	switch algorithm {
	case TOTPAlgorithmSHA1, TOTPAlgorithmSHA256, TOTPAlgorithmSHA512:
		return algorithm, nil
	default:
		return "", errors.New("unknown TOTP algorithm")
	}
}

type EncryptedSecret []byte

func NewEncryptedSecret(value []byte) (EncryptedSecret, error) {
	if len(value) == 0 {
		return nil, errors.New("encrypted TOTP secret is required")
	}

	return EncryptedSecret(bytes.Clone(value)), nil
}

type EncryptionKeyRef string

func NewEncryptionKeyRef(value string) (EncryptionKeyRef, error) {
	value, ok := domain.NormalizeRequiredText(value)
	if !ok {
		return "", errors.New("TOTP encryption key reference is required")
	}

	return EncryptionKeyRef(value), nil
}

type CredentialID []byte

func NewCredentialID(value []byte) (CredentialID, error) {
	if len(value) == 0 {
		return nil, errors.New("WebAuthn credential identifier is required")
	}

	return CredentialID(bytes.Clone(value)), nil
}

type PublicKey []byte

func NewPublicKey(value []byte) (PublicKey, error) {
	if len(value) == 0 {
		return nil, errors.New("WebAuthn public key is required")
	}

	return PublicKey(bytes.Clone(value)), nil
}

type Transport string

func NewTransport(value string) (Transport, error) {
	value, ok := domain.NormalizeToken(strings.ToLower(value))
	if !ok {
		return "", errors.New("WebAuthn transport must be a non-empty token")
	}

	return Transport(value), nil
}
