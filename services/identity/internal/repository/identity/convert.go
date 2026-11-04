package identity

import (
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

func toDomain(
	identity row,
	emails []emailRow,
	phones []phoneRow,
	accounts []externalAccountRow,
) *identitydomain.Identity {
	entity := &identitydomain.Identity{
		ID:        identitydomain.ID(identity.ID),
		Status:    identitydomain.Status(identity.Status),
		CreatedAt: identity.CreatedAt,
		UpdatedAt: identity.UpdatedAt,
		DeletedAt: domain.ClonePointer(identity.DeletedAt),
		Emails:    make([]identitydomain.Email, 0, len(emails)),
		Phones:    make([]identitydomain.Phone, 0, len(phones)),
		ExternalAccounts: make(
			[]identitydomain.ExternalAccount,
			0,
			len(accounts),
		),
	}

	for _, email := range emails {
		entity.Emails = append(entity.Emails, identitydomain.Email{
			ID:         identitydomain.EmailID(email.ID),
			Address:    identitydomain.EmailAddress(email.Email),
			IsPrimary:  email.IsPrimary,
			VerifiedAt: domain.ClonePointer(email.VerifiedAt),
			CreatedAt:  email.CreatedAt,
			UpdatedAt:  email.UpdatedAt,
			DeletedAt:  domain.ClonePointer(email.DeletedAt),
		})
	}

	for _, phone := range phones {
		entity.Phones = append(entity.Phones, identitydomain.Phone{
			ID:         identitydomain.PhoneID(phone.ID),
			Number:     identitydomain.PhoneNumber(phone.Phone),
			IsPrimary:  phone.IsPrimary,
			VerifiedAt: domain.ClonePointer(phone.VerifiedAt),
			CreatedAt:  phone.CreatedAt,
			UpdatedAt:  phone.UpdatedAt,
			DeletedAt:  domain.ClonePointer(phone.DeletedAt),
		})
	}

	for _, account := range accounts {
		entity.ExternalAccounts = append(entity.ExternalAccounts, identitydomain.ExternalAccount{
			ID:                  identitydomain.ExternalAccountID(account.ID),
			Provider:            identitydomain.ExternalProvider(account.Provider),
			Subject:             identitydomain.ExternalSubject(account.Subject),
			LinkedAt:            account.LinkedAt,
			LastAuthenticatedAt: domain.ClonePointer(account.LastAuthenticatedAt),
			DisabledAt:          domain.ClonePointer(account.DisabledAt),
			DeletedAt:           domain.ClonePointer(account.DeletedAt),
		})
	}

	return entity
}
