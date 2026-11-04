package serviceaccount

import (
	"github.com/google/uuid"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	accountdomain "github.com/krackl1n/careeria/services/identity/internal/domain/serviceaccount"
)

func toDomain(m row, credentials []credentialRow) *accountdomain.ServiceAccount {
	e := &accountdomain.ServiceAccount{
		ID:                  accountdomain.ID(m.ID),
		OrganizationID:      accountdomain.OrganizationID(m.OrganizationID),
		AccountKey:          accountdomain.AccountKey(m.AccountKey),
		Name:                m.Name,
		Description:         domain.ClonePointer(m.Description),
		Status:              accountdomain.Status(m.Status),
		CreatedByIdentityID: identityIDToDomain(m.CreatedByIdentityID),
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
		DisabledAt:          domain.ClonePointer(m.DisabledAt),
		DeletedAt:           domain.ClonePointer(m.DeletedAt),
	}

	for _, v := range credentials {
		e.Credentials = append(
			e.Credentials,
			accountdomain.Credential{
				ID:                  accountdomain.CredentialID(v.ID),
				Type:                accountdomain.CredentialType(v.Type),
				SecretHash:          accountdomain.SecretHash(v.SecretHash),
				SecretPrefix:        secretPrefixToDomain(v.SecretPrefix),
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

func credentialToRow(
	accountID accountdomain.ID,
	v accountdomain.Credential,
) credentialRow {
	return credentialRow{
		ID:                  v.ID.UUID(),
		ServiceAccountID:    accountID.UUID(),
		Type:                string(v.Type),
		SecretHash:          string(v.SecretHash),
		SecretPrefix:        secretPrefixToRow(v.SecretPrefix),
		Name:                domain.ClonePointer(v.Name),
		CreatedByIdentityID: identityIDToRow(v.CreatedByIdentityID),
		CreatedAt:           v.CreatedAt,
		UpdatedAt:           v.UpdatedAt,
		ExpiresAt:           domain.ClonePointer(v.ExpiresAt),
		RevokedAt:           domain.ClonePointer(v.RevokedAt),
		RevokeReason:        domain.ClonePointer(v.RevokeReason),
		LastUsedAt:          domain.ClonePointer(v.LastUsedAt),
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

func secretPrefixToDomain(v *string) *accountdomain.SecretPrefix {
	if v == nil {
		return nil
	}

	x := accountdomain.SecretPrefix(*v)
	return &x
}

func secretPrefixToRow(v *accountdomain.SecretPrefix) *string {
	if v == nil {
		return nil
	}

	x := string(*v)
	return &x
}
