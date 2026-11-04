package session

import (
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	authenticationdomain "github.com/krackl1n/careeria/services/identity/internal/domain/authentication"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauth"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

func toDomain(model row) *sessiondomain.Session {
	methods := make([]authenticationdomain.Method, 0, len(model.AuthenticatedMethods))
	for _, value := range model.AuthenticatedMethods {
		methods = append(methods, authenticationdomain.Method(value))
	}

	scopes := make([]oauthdomain.Scope, 0, len(model.Scopes))
	for _, value := range model.Scopes {
		scopes = append(scopes, oauthdomain.Scope(value))
	}

	return &sessiondomain.Session{
		ID:                   sessiondomain.ID(model.ID),
		IdentityID:           identitydomain.ID(model.IdentityID),
		OAuthClientID:        oauthclientdomain.ID(model.OAuthClientID),
		AuthenticationLevel:  authenticationdomain.Level(model.AuthenticationLevel),
		AuthenticatedMethods: methods,
		Scopes:               scopes,
		IPAddress:            ipAddressToDomain(model.IPAddress),
		UserAgent:            domain.ClonePointer(model.UserAgent),
		DeviceName:           domain.ClonePointer(model.DeviceName),
		CreatedAt:            model.CreatedAt,
		AuthenticatedAt:      model.AuthenticatedAt,
		LastSeenAt:           domain.ClonePointer(model.LastSeenAt),
		IdleExpiresAt:        model.IdleExpiresAt,
		AbsoluteExpiresAt:    model.AbsoluteExpiresAt,
		RevokedAt:            domain.ClonePointer(model.RevokedAt),
		RevokeReason:         domain.ClonePointer(model.RevokeReason),
	}
}

func toRow(entity *sessiondomain.Session) row {
	methods := make([]string, 0, len(entity.AuthenticatedMethods))
	for _, value := range entity.AuthenticatedMethods {
		methods = append(methods, string(value))
	}

	scopes := make([]string, 0, len(entity.Scopes))
	for _, value := range entity.Scopes {
		scopes = append(scopes, string(value))
	}

	return row{
		ID:                   entity.ID.UUID(),
		IdentityID:           entity.IdentityID.UUID(),
		OAuthClientID:        entity.OAuthClientID.UUID(),
		AuthenticationLevel:  string(entity.AuthenticationLevel),
		AuthenticatedMethods: methods,
		Scopes:               scopes,
		IPAddress:            ipAddressToRow(entity.IPAddress),
		UserAgent:            domain.ClonePointer(entity.UserAgent),
		DeviceName:           domain.ClonePointer(entity.DeviceName),
		CreatedAt:            entity.CreatedAt,
		AuthenticatedAt:      entity.AuthenticatedAt,
		LastSeenAt:           domain.ClonePointer(entity.LastSeenAt),
		IdleExpiresAt:        entity.IdleExpiresAt,
		AbsoluteExpiresAt:    entity.AbsoluteExpiresAt,
		RevokedAt:            domain.ClonePointer(entity.RevokedAt),
		RevokeReason:         domain.ClonePointer(entity.RevokeReason),
	}
}

func ipAddressToDomain(value *string) *sessiondomain.IPAddress {
	if value == nil {
		return nil
	}

	address := sessiondomain.IPAddress(*value)
	return &address
}

func ipAddressToRow(value *sessiondomain.IPAddress) *string {
	if value == nil {
		return nil
	}

	address := string(*value)
	return &address
}
