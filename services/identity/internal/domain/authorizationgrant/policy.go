package authorizationgrant

import (
	"errors"
	"slices"
	"time"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
)

type ConsentDecision string

const (
	ConsentRequired        ConsentDecision = "required"
	ConsentRememberedGrant ConsentDecision = "remembered_grant"
	ConsentTrustedClient   ConsentDecision = "trusted_client"
)

type Decision struct {
	Kind   ConsentDecision
	Scopes []oauthdomain.Scope
}

func ResolveConsent(
	client *oauthclientdomain.OAuthClient,
	grant *AuthorizationGrant,
	requested []oauthdomain.Scope,
	forceConsent bool,
	now time.Time,
) (Decision, error) {
	if client == nil {
		return Decision{}, errors.New("resolve consent: OAuth client is nil")
	}
	if err := validateScopes(requested); err != nil {
		return Decision{}, err
	}
	if domain.HasDuplicates(requested) || !clientAllowsScopes(client, requested) {
		return Decision{}, errors.New("resolve consent: requested scope is not allowed for OAuth client")
	}

	decision := Decision{Kind: ConsentRequired, Scopes: slices.Clone(requested)}
	if forceConsent || client.ConsentMode == oauthclientdomain.ConsentModeRequired {
		return decision, nil
	}
	if client.ConsentMode == oauthclientdomain.ConsentModeTrusted {
		decision.Kind = ConsentTrustedClient
		return decision, nil
	}
	if client.ConsentMode == oauthclientdomain.ConsentModeRemember && grant != nil && grant.Covers(requested, now) {
		decision.Kind = ConsentRememberedGrant
	}
	return decision, nil
}

func clientAllowsScopes(
	client *oauthclientdomain.OAuthClient,
	requested []oauthdomain.Scope,
) bool {
	allowed := make(map[string]struct{}, len(client.Scopes))
	for _, registration := range client.Scopes {
		allowed[string(registration.Scope)] = struct{}{}
	}
	for _, scope := range requested {
		if _, ok := allowed[string(scope)]; !ok {
			return false
		}
	}
	return true
}
