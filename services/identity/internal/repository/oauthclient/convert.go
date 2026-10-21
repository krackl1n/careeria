package oauthclient

import (
	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthvaluedomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
)

func toDomain(
	m row,
	redirects []redirectURIRow,
	postLogoutRedirects []redirectURIRow,
	grantTypes []registrationRow,
	scopes []registrationRow,
	credentials []credentialRow,
) *oauthdomain.OAuthClient {
	e := &oauthdomain.OAuthClient{
		ID:                      oauthdomain.ID(m.ID),
		ClientID:                oauthdomain.ClientID(m.ClientID),
		ClientType:              oauthdomain.ClientType(m.ClientType),
		TokenEndpointAuthMethod: oauthdomain.TokenEndpointAuthMethod(m.TokenEndpointAuthMethod),
		Name:                    m.Name,
		Description:             domain.ClonePointer(m.Description),
		RequirePKCE:             m.RequirePKCE,
		ConsentMode:             oauthdomain.ConsentMode(m.ConsentMode),
		Status:                  oauthdomain.Status(m.Status),
		CreatedByIdentityID:     identityIDToDomain(m.CreatedByIdentityID),
		CreatedAt:               m.CreatedAt,
		UpdatedAt:               m.UpdatedAt,
		DisabledAt:              domain.ClonePointer(m.DisabledAt),
		DeletedAt:               domain.ClonePointer(m.DeletedAt),
	}

	for _, v := range redirects {
		e.RedirectURIs = append(
			e.RedirectURIs,
			oauthdomain.RedirectURI{
				ID:        oauthdomain.RedirectURIID(v.ID),
				URI:       oauthdomain.RedirectURIValue(v.URI),
				CreatedAt: v.CreatedAt,
			},
		)
	}

	for _, v := range postLogoutRedirects {
		e.PostLogoutRedirectURIs = append(
			e.PostLogoutRedirectURIs,
			oauthdomain.PostLogoutRedirectURI{
				ID:        oauthdomain.PostLogoutRedirectURIID(v.ID),
				URI:       oauthdomain.PostLogoutRedirectURIValue(v.URI),
				CreatedAt: v.CreatedAt,
			},
		)
	}

	for _, v := range grantTypes {
		e.GrantTypes = append(
			e.GrantTypes,
			oauthdomain.GrantTypeRegistration{
				GrantType: oauthdomain.GrantType(v.Value),
				CreatedAt: v.CreatedAt,
			},
		)
	}

	for _, v := range scopes {
		e.Scopes = append(
			e.Scopes,
			oauthdomain.ScopeRegistration{
				Scope:     oauthvaluedomain.Scope(v.Value),
				CreatedAt: v.CreatedAt,
			},
		)
	}

	for _, v := range credentials {
		e.Credentials = append(
			e.Credentials,
			oauthdomain.Credential{
				ID:                  oauthdomain.CredentialID(v.ID),
				SecretHash:          oauthdomain.SecretHash(v.SecretHash),
				Name:                domain.ClonePointer(v.Name),
				CreatedByIdentityID: identityIDToDomain(v.CreatedByIdentityID),
				CreatedAt:           v.CreatedAt,
				UpdatedAt:           v.UpdatedAt,
				ExpiresAt:           domain.ClonePointer(v.ExpiresAt),
				RevokedAt:           domain.ClonePointer(v.RevokedAt),
				RevokeReason:        domain.ClonePointer(v.RevokeReason),
				LastUsedAt:          domain.ClonePointer(v.LastUsedAt),
			},
		)
	}

	return e
}

func identityIDToDomain(v *uuid.UUID) *identitydomain.ID {
	if v == nil {
		return nil
	}

	x := identitydomain.ID(*v)
	return &x
}

func identityIDToRow(v *identitydomain.ID) *uuid.UUID {
	if v == nil {
		return nil
	}

	x := v.UUID()
	return &x
}
