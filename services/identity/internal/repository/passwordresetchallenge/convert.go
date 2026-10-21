package passwordresetchallenge

import (
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	challenge "github.com/krackl1n/careeria/services/identity/internal/domain/passwordresetchallenge"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

func toDomain(model row) *challenge.PasswordResetChallenge {
	return &challenge.PasswordResetChallenge{
		ID:              challenge.ID(model.ID),
		IdentityID:      identitydomain.ID(model.IdentityID),
		TokenHash:       challenge.TokenHash(model.TokenHash),
		Attempts:        model.Attempts,
		MaxAttempts:     model.MaxAttempts,
		RequestedFromIP: ipAddressToDomain(model.RequestedFromIP),
		ConsumedFromIP:  ipAddressToDomain(model.ConsumedFromIP),
		CreatedAt:       model.CreatedAt,
		ExpiresAt:       model.ExpiresAt,
		LastAttemptAt:   domain.ClonePointer(model.LastAttemptAt),
		ConsumedAt:      domain.ClonePointer(model.ConsumedAt),
		RevokedAt:       domain.ClonePointer(model.RevokedAt),
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
