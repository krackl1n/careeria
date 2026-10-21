package oauthclient

import (
	"fmt"
	"time"

	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
)

func (c *OAuthClient) ValidateAuthorizationRedirectURI(
	redirectURI RedirectURIValue,
) error {
	if c == nil {
		return fmt.Errorf("validate OAuth authorization redirect URI: OAuth client is nil")
	}
	if c.redirectURIIndexByValue(redirectURI) < 0 {
		return fmt.Errorf("validate OAuth authorization redirect URI: %w", ErrRedirectURINotFound)
	}

	return nil
}

func (c *OAuthClient) ValidateAuthorizationCodeFlow(
	scopes []oauthdomain.Scope,
	now time.Time,
) error {
	if err := c.ensureActive(now); err != nil {
		return fmt.Errorf("validate OAuth authorization code flow: %w", err)
	}
	if c.grantTypeIndex(GrantTypeAuthorizationCode) < 0 {
		return fmt.Errorf("validate OAuth authorization code flow: %w", ErrAuthorizationCodeFlowNotAllowed)
	}
	for _, scope := range scopes {
		if c.scopeIndex(scope) < 0 {
			return fmt.Errorf("validate OAuth authorization code flow: %w", ErrScopeNotFound)
		}
	}

	return nil
}

func (c *OAuthClient) ValidateTokenExchange(grantType GrantType, now time.Time) error {
	if err := c.ensureActive(now); err != nil {
		return fmt.Errorf("validate OAuth token exchange: %w", err)
	}
	if c.grantTypeIndex(grantType) < 0 {
		return fmt.Errorf("validate OAuth token exchange: %w", ErrGrantTypeNotFound)
	}

	return nil
}

func (c *OAuthClient) AllowsGrantType(grantType GrantType) bool {
	return c != nil && c.grantTypeIndex(grantType) >= 0
}
