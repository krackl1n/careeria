package authorizationgrant

import (
	"errors"
	"fmt"
	"slices"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
)

type AuthorizationGrant struct {
	ID            ID
	IdentityID    identitydomain.ID
	OAuthClientID oauthclientdomain.ID
	Source        Source
	Scopes        []oauthdomain.Scope

	GrantedAt time.Time
	UpdatedAt time.Time
	ExpiresAt *time.Time
	RevokedAt *time.Time
}

func New(
	id ID,
	identityID identitydomain.ID,
	oauthClientID oauthclientdomain.ID,
	source Source,
	scopes []oauthdomain.Scope,
	expiresAt *time.Time,
	now time.Time,
) (*AuthorizationGrant, error) {
	if !id.IsValid() || !identityID.IsValid() || !oauthClientID.IsValid() {
		return nil, errors.New("create authorization grant: identifier is invalid")
	}
	if _, err := ParseSource(string(source)); err != nil {
		return nil, fmt.Errorf("create authorization grant: %w", err)
	}
	if err := validateScopes(scopes); err != nil {
		return nil, fmt.Errorf("create authorization grant: %w", err)
	}
	if now.IsZero() || expiresAt != nil && !expiresAt.After(now) {
		return nil, errors.New("create authorization grant: expiration is invalid")
	}

	return &AuthorizationGrant{
		ID:            id,
		IdentityID:    identityID,
		OAuthClientID: oauthClientID,
		Source:        source,
		Scopes:        slices.Clone(scopes),
		GrantedAt:     now,
		UpdatedAt:     now,
		ExpiresAt:     domain.ClonePointer(expiresAt),
	}, nil
}

func (g *AuthorizationGrant) Covers(scopes []oauthdomain.Scope, now time.Time) bool {
	if g == nil || g.validateActive(now) != nil || len(scopes) == 0 {
		return false
	}
	granted := make(map[oauthdomain.Scope]struct{}, len(g.Scopes))
	for _, scope := range g.Scopes {
		granted[scope] = struct{}{}
	}
	for _, scope := range scopes {
		if _, ok := granted[scope]; !ok {
			return false
		}
	}
	return true
}

func (g *AuthorizationGrant) Extend(
	scopes []oauthdomain.Scope,
	expiresAt *time.Time,
	now time.Time,
) error {
	if err := g.validateActive(now); err != nil {
		return fmt.Errorf("extend authorization grant: %w", err)
	}
	if err := validateScopes(scopes); err != nil {
		return fmt.Errorf("extend authorization grant: %w", err)
	}
	if expiresAt != nil && !expiresAt.After(now) {
		return errors.New("extend authorization grant: expiration is invalid")
	}

	seen := make(map[oauthdomain.Scope]struct{}, len(g.Scopes)+len(scopes))
	merged := make([]oauthdomain.Scope, 0, len(g.Scopes)+len(scopes))
	for _, scope := range append(slices.Clone(g.Scopes), scopes...) {
		if _, exists := seen[scope]; exists {
			continue
		}
		seen[scope] = struct{}{}
		merged = append(merged, scope)
	}

	g.Scopes = merged
	g.ExpiresAt = domain.ClonePointer(expiresAt)
	g.UpdatedAt = now
	return nil
}

func (g *AuthorizationGrant) Replace(
	source Source,
	scopes []oauthdomain.Scope,
	expiresAt *time.Time,
	now time.Time,
) error {
	if g == nil {
		return errors.New("replace authorization grant: grant is nil")
	}
	if _, err := ParseSource(string(source)); err != nil {
		return fmt.Errorf("replace authorization grant: %w", err)
	}
	if err := validateScopes(scopes); err != nil {
		return fmt.Errorf("replace authorization grant: %w", err)
	}
	if now.IsZero() || now.Before(g.UpdatedAt) || expiresAt != nil && !expiresAt.After(now) {
		return errors.New("replace authorization grant: time is invalid")
	}

	g.Source = source
	g.Scopes = slices.Clone(scopes)
	g.GrantedAt = now
	g.UpdatedAt = now
	g.ExpiresAt = domain.ClonePointer(expiresAt)
	g.RevokedAt = nil
	return nil
}

func (g *AuthorizationGrant) Revoke(now time.Time) error {
	if g == nil {
		return errors.New("revoke authorization grant: grant is nil")
	}
	if g.RevokedAt != nil {
		return nil
	}
	if now.IsZero() || now.Before(g.UpdatedAt) {
		return errors.New("revoke authorization grant: time is invalid")
	}

	g.RevokedAt = domain.ClonePointer(&now)
	g.UpdatedAt = now
	return nil
}

func (g *AuthorizationGrant) validateActive(now time.Time) error {
	if g == nil {
		return errors.New("authorization grant is nil")
	}
	if now.IsZero() || now.Before(g.UpdatedAt) {
		return errors.New("authorization grant use time is invalid")
	}
	if g.RevokedAt != nil {
		return ErrRevoked
	}
	if g.ExpiresAt != nil && !now.Before(*g.ExpiresAt) {
		return ErrExpired
	}
	return nil
}

func validateScopes(scopes []oauthdomain.Scope) error {
	if len(scopes) == 0 {
		return errors.New("at least one authorization scope is required")
	}
	if domain.HasDuplicates(scopes) {
		return errors.New("authorization scope is duplicated")
	}
	for _, scope := range scopes {
		if scope == "" {
			return errors.New("authorization scope is empty")
		}
	}
	return nil
}
