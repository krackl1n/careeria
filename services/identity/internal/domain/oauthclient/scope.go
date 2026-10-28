package oauthclient

import (
	"time"

	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
)

type ScopeRegistration struct {
	Scope oauthdomain.Scope

	CreatedAt time.Time
}

func newScopeRegistration(scope oauthdomain.Scope, now time.Time) ScopeRegistration {
	return ScopeRegistration{Scope: scope, CreatedAt: now}
}
