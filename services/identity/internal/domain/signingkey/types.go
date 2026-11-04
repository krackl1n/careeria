package signingkey

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
)

var privateJWKMembers = [...]string{"d", "p", "q", "dp", "dq", "qi", "oth", "k"}

type ID uuid.UUID

func (id ID) UUID() uuid.UUID { return uuid.UUID(id) }

func (id ID) IsValid() bool { return uuid.UUID(id) != uuid.Nil }

type KeyID string

func NewKeyID(value string) (KeyID, error) {
	value, ok := domain.NormalizeToken(value)
	if !ok {
		return "", errors.New("signing key ID must be a non-empty token")
	}

	return KeyID(value), nil
}

type Algorithm string

const (
	AlgorithmEdDSA Algorithm = "EdDSA"
	AlgorithmES256 Algorithm = "ES256"
	AlgorithmRS256 Algorithm = "RS256"
)

func NewAlgorithm(value string) (Algorithm, error) {
	value, ok := domain.NormalizeToken(value)
	if !ok {
		return "", errors.New("signing key algorithm must be a non-empty token")
	}

	algorithm := Algorithm(value)
	switch algorithm {
	case AlgorithmEdDSA, AlgorithmES256, AlgorithmRS256:
		return algorithm, nil
	default:
		return "", errors.New("unsupported signing key algorithm")
	}
}

type PrivateKeyRef string

func NewPrivateKeyRef(value string) (PrivateKeyRef, error) {
	value, ok := domain.NormalizeRequiredText(value)
	if !ok {
		return "", errors.New("signing private key reference is required")
	}

	return PrivateKeyRef(value), nil
}

type PublicJWK []byte

func NewPublicJWK(value []byte) (PublicJWK, error) {
	if err := validatePublicJWK(value); err != nil {
		return nil, fmt.Errorf("create public JWK: %w", err)
	}

	return PublicJWK(slices.Clone(value)), nil
}

type Status string

const (
	StatusPending     Status = "pending"
	StatusActive      Status = "active"
	StatusRetiring    Status = "retiring"
	StatusRetired     Status = "retired"
	StatusCompromised Status = "compromised"
)

func ParseStatus(value string) (Status, error) {
	status := Status(value)
	switch status {
	case StatusPending, StatusActive, StatusRetiring, StatusRetired, StatusCompromised:
		return status, nil
	default:
		return "", errors.New("unknown signing key status")
	}
}

func validatePublicJWK(value []byte) error {
	value = bytes.TrimSpace(value)
	if len(value) == 0 || value[0] != '{' {
		return errors.New("public JWK must be a JSON object")
	}

	var members map[string]json.RawMessage
	if err := json.Unmarshal(value, &members); err != nil {
		return fmt.Errorf("decode public JWK: %w", err)
	}
	if members == nil {
		return errors.New("public JWK must be a JSON object")
	}
	for _, member := range privateJWKMembers {
		if _, exists := members[member]; exists {
			return fmt.Errorf("public JWK contains private member %q", member)
		}
	}

	var keyType string
	if raw, exists := members["kty"]; !exists || json.Unmarshal(raw, &keyType) != nil || strings.TrimSpace(keyType) == "" {
		return errors.New("public JWK key type is required")
	}

	return nil
}

func validatePublicJWKMetadata(value []byte, keyID KeyID, algorithm Algorithm) error {
	var members map[string]json.RawMessage
	if err := json.Unmarshal(value, &members); err != nil {
		return fmt.Errorf("decode public JWK metadata: %w", err)
	}
	kid, err := requiredJWKString(members, "kid")
	if err != nil {
		return err
	}
	if kid != string(keyID) {
		return errors.New("public JWK member \"kid\" does not match signing key metadata")
	}
	for member, expected := range map[string]string{"alg": string(algorithm), "use": "sig"} {
		if raw, exists := members[member]; exists {
			var actual string
			if err := json.Unmarshal(raw, &actual); err != nil || actual != expected {
				return fmt.Errorf("public JWK member %q does not match signing key metadata", member)
			}
		}
	}

	keyType, err := requiredJWKString(members, "kty")
	if err != nil {
		return err
	}
	switch algorithm {
	case AlgorithmEdDSA:
		curve, err := requiredJWKString(members, "crv")
		if err != nil {
			return err
		}
		if keyType != "OKP" || curve != "Ed25519" && curve != "Ed448" {
			return errors.New("EdDSA public JWK must use an Ed25519 or Ed448 OKP key")
		}
		if _, err := requiredJWKString(members, "x"); err != nil {
			return err
		}
	case AlgorithmES256:
		curve, err := requiredJWKString(members, "crv")
		if err != nil {
			return err
		}
		if keyType != "EC" || curve != "P-256" {
			return errors.New("ES256 public JWK must use a P-256 EC key")
		}
		for _, member := range []string{"x", "y"} {
			if _, err := requiredJWKString(members, member); err != nil {
				return err
			}
		}
	case AlgorithmRS256:
		if keyType != "RSA" {
			return errors.New("RS256 public JWK must use an RSA key")
		}
		for _, member := range []string{"n", "e"} {
			if _, err := requiredJWKString(members, member); err != nil {
				return err
			}
		}
	default:
		return errors.New("unsupported public JWK algorithm")
	}

	return nil
}

func requiredJWKString(members map[string]json.RawMessage, member string) (string, error) {
	raw, exists := members[member]
	if !exists {
		return "", fmt.Errorf("public JWK member %q is required", member)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("public JWK member %q must be a non-empty string", member)
	}

	return value, nil
}
