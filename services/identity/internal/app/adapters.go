package app

import (
	"fmt"
	"net/url"

	transactiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	passworddomain "github.com/krackl1n/careeria/services/identity/internal/domain/password"
	"github.com/krackl1n/careeria/services/identity/pkg/password"
	"github.com/krackl1n/careeria/services/identity/pkg/token"
)

type passwordHasher struct{ value *password.Argon2id }

func (h passwordHasher) Hash(value string) (string, error) { return h.value.Hash(value) }

type passwordVerifier struct{ value *password.Argon2id }

func (v passwordVerifier) Verify(value string, hash passworddomain.Hash) error {
	valid, err := v.value.Verify(value, string(hash))
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("password does not match")
	}
	return nil
}

type clientSecretVerifier struct{ value *password.Argon2id }

func (v clientSecretVerifier) Verify(value, hash string) (bool, error) {
	return v.value.Verify(value, hash)
}

type tokenGenerator struct {
	generator *token.Random
	digester  *token.SHA256
}

func (g tokenGenerator) New() (string, string, error) {
	value, err := g.generator.Generate()
	if err != nil {
		return "", "", err
	}
	return value, g.digester.Digest(value), nil
}
func (g tokenGenerator) Hash(value string) string { return g.digester.Digest(value) }

type authenticationRedirect struct{ baseURL string }

func (r authenticationRedirect) BuildAuthorizationRedirect(id transactiondomain.ID) (string, error) {
	value, err := url.Parse(r.baseURL)
	if err != nil {
		return "", fmt.Errorf("parse authentication redirect URI: %w", err)
	}
	query := value.Query()
	query.Set("transaction_id", id.UUID().String())
	value.RawQuery = query.Encode()
	return value.String(), nil
}
