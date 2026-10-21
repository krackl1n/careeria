package authorizationgrant

import (
	"github.com/krackl1n/careeria/services/identity/internal/domain"
	grantdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationgrant"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
)

func toDomain(grant row, scopes []scopeRow) *grantdomain.AuthorizationGrant {
	values := make([]oauthdomain.Scope, 0, len(scopes))
	for _, scope := range scopes {
		values = append(values, oauthdomain.Scope(scope.Scope))
	}

	return &grantdomain.AuthorizationGrant{
		ID:            grantdomain.ID(grant.ID),
		IdentityID:    identitydomain.ID(grant.IdentityID),
		OAuthClientID: oauthclientdomain.ID(grant.OAuthClientID),
		Source:        grantdomain.Source(grant.Source),
		Scopes:        values,
		GrantedAt:     grant.GrantedAt,
		UpdatedAt:     grant.UpdatedAt,
		ExpiresAt:     domain.ClonePointer(grant.ExpiresAt),
		RevokedAt:     domain.ClonePointer(grant.RevokedAt),
	}
}
