package jwt

import (
	"errors"
	"strings"
)

type Config struct {
	KeyID         string
	PrivateKeyRef string
	KeyFile       string
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.KeyID) == "" {
		return errors.New("key ID is required")
	}
	if strings.TrimSpace(c.PrivateKeyRef) == "" {
		return errors.New("private key reference is required")
	}
	if strings.TrimSpace(c.KeyFile) == "" {
		return errors.New("key file is required")
	}

	return nil
}
