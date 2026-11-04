package accesstoken

import (
	"errors"
	"strings"
	"time"

	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
)

type Config struct {
	Issuer    string
	Audience  []string
	Algorithm string
	Lifetime  time.Duration
}

func (c Config) Validate() error {
	if c.Issuer == "" || strings.TrimSpace(c.Issuer) != c.Issuer {
		return errors.New("issuer is invalid")
	}
	if len(c.Audience) == 0 {
		return errors.New("audience is required")
	}
	for _, audience := range c.Audience {
		if audience == "" || strings.TrimSpace(audience) != audience {
			return errors.New("audience is invalid")
		}
	}
	if c.Algorithm == "" || strings.TrimSpace(c.Algorithm) != c.Algorithm {
		return errors.New("algorithm is invalid")
	}
	if _, err := signingkeydomain.NewAlgorithm(c.Algorithm); err != nil {
		return errors.New("algorithm is invalid")
	}
	if c.Lifetime < time.Second {
		return errors.New("lifetime must be at least one second")
	}
	if c.Lifetime%time.Second != 0 {
		return errors.New("lifetime must be a whole number of seconds")
	}

	return nil
}
