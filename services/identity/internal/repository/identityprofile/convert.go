package identityprofile

import (
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	identityprofiledomain "github.com/krackl1n/careeria/services/identity/internal/domain/identityprofile"
)

func toDomain(model row) *identityprofiledomain.Profile {
	return &identityprofiledomain.Profile{
		IdentityID:     identitydomain.ID(model.IdentityID),
		FamilyName:     model.FamilyName,
		GivenName:      model.GivenName,
		MiddleName:     domain.ClonePointer(model.MiddleName),
		Source:         identityprofiledomain.Source(model.Source),
		AssuranceLevel: identityprofiledomain.AssuranceLevel(model.AssuranceLevel),
		Version:        model.Version,
		CreatedAt:      model.CreatedAt,
		UpdatedAt:      model.UpdatedAt,
	}
}
