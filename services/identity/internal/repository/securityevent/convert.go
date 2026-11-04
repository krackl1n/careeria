package securityevent

import (
	"slices"

	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/securityevent"
	serviceaccountdomain "github.com/krackl1n/careeria/services/identity/internal/domain/serviceaccount"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

func toDomain(m row) *eventdomain.SecurityEvent {
	return &eventdomain.SecurityEvent{
		ID:     eventdomain.ID(m.ID),
		Type:   eventdomain.EventType(m.EventType),
		Result: eventdomain.Result(m.Result),
		Actor: eventdomain.Actor{
			Type:             eventdomain.ActorType(m.ActorType),
			IdentityID:       identityIDToDomain(m.IdentityID),
			ServiceAccountID: serviceAccountIDToDomain(m.ServiceAccountID),
		},
		SessionID:      sessionIDToDomain(m.SessionID),
		OAuthClientID:  oauthIDToDomain(m.OAuthClientID),
		IdentifierHash: eventdomain.IdentifierHash(slices.Clone(m.IdentifierHash)),
		IPAddress:      ipToDomain(m.IPAddress),
		UserAgent:      domain.ClonePointer(m.UserAgent),
		CorrelationID:  domain.ClonePointer(m.CorrelationID),
		Metadata:       eventdomain.Metadata(slices.Clone(m.Metadata)),
		OccurredAt:     m.OccurredAt,
	}
}

func toRow(e *eventdomain.SecurityEvent) row {
	return row{
		ID:               e.ID.UUID(),
		EventType:        string(e.Type),
		Result:           string(e.Result),
		ActorType:        string(e.Actor.Type),
		IdentityID:       identityIDToRow(e.Actor.IdentityID),
		ServiceAccountID: serviceAccountIDToRow(e.Actor.ServiceAccountID),
		SessionID:        sessionIDToRow(e.SessionID),
		OAuthClientID:    oauthIDToRow(e.OAuthClientID),
		IdentifierHash:   slices.Clone(e.IdentifierHash),
		IPAddress:        ipToRow(e.IPAddress),
		UserAgent:        domain.ClonePointer(e.UserAgent),
		CorrelationID:    domain.ClonePointer(e.CorrelationID),
		OccurredAt:       e.OccurredAt,
		Metadata:         slices.Clone(e.Metadata),
	}
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

func serviceAccountIDToDomain(v *uuid.UUID) *serviceaccountdomain.ID {
	if v == nil {
		return nil
	}

	x := serviceaccountdomain.ID(*v)
	return &x
}

func serviceAccountIDToRow(v *serviceaccountdomain.ID) *uuid.UUID {
	if v == nil {
		return nil
	}

	x := v.UUID()
	return &x
}

func sessionIDToDomain(v *uuid.UUID) *sessiondomain.ID {
	if v == nil {
		return nil
	}

	x := sessiondomain.ID(*v)
	return &x
}

func sessionIDToRow(v *sessiondomain.ID) *uuid.UUID {
	if v == nil {
		return nil
	}

	x := v.UUID()
	return &x
}

func oauthIDToDomain(v *uuid.UUID) *oauthdomain.ID {
	if v == nil {
		return nil
	}

	x := oauthdomain.ID(*v)
	return &x
}

func oauthIDToRow(v *oauthdomain.ID) *uuid.UUID {
	if v == nil {
		return nil
	}

	x := v.UUID()
	return &x
}

func ipToDomain(v *string) *sessiondomain.IPAddress {
	if v == nil {
		return nil
	}

	x := sessiondomain.IPAddress(*v)
	return &x
}

func ipToRow(v *sessiondomain.IPAddress) *string {
	if v == nil {
		return nil
	}

	x := string(*v)
	return &x
}
