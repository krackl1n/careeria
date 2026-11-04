package metadata

import (
	"errors"
	"strings"
)

type Config struct {
	Issuer                  string
	IDTokenSigningAlgorithm string
}

func (c Config) Validate() error {
	if c.Issuer == "" || strings.TrimSpace(c.Issuer) != c.Issuer {
		return errors.New("issuer is invalid")
	}
	if c.IDTokenSigningAlgorithm == "" || strings.TrimSpace(c.IDTokenSigningAlgorithm) != c.IDTokenSigningAlgorithm {
		return errors.New("ID token signing algorithm is invalid")
	}

	return nil
}
