package idtoken

import (
	"errors"
	"strings"
	"time"

	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
)

type Config struct {
	Issuer    string
	Algorithm string
	Lifetime  time.Duration
}

func (c Config) Validate() error {
	if c.Algorithm == "" || strings.TrimSpace(c.Algorithm) != c.Algorithm {
		return errors.New("algorithm is invalid")
	}
	if _, err := signingkeydomain.NewAlgorithm(c.Algorithm); err != nil {
		return errors.New("algorithm is invalid")
	}
	if c.Lifetime < time.Second || c.Lifetime%time.Second != 0 {
		return errors.New("lifetime must be a positive whole number of seconds")
	}

	return nil
}
