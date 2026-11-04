package authn

import (
	"errors"
	"fmt"
	"strings"
)

type AuthConfig struct {
	Issuer    string
	Audience  []string
	JWKSCache JWKSCacheConfig
}

func (c AuthConfig) Validate() error {
	if strings.TrimSpace(c.Issuer) == "" {
		return errors.New("JWT issuer is required")
	}
	if len(c.Audience) == 0 {
		return errors.New("JWT audience is required")
	}
	for _, audience := range c.Audience {
		if strings.TrimSpace(audience) == "" {
			return errors.New("JWT audience must not be empty")
		}
	}
	if err := c.JWKSCache.Validate(); err != nil {
		return fmt.Errorf("validate JWKS cache config: %w", err)
	}
	return nil
}
