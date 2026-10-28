package recoverycodeset

import (
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	setdomain "github.com/krackl1n/careeria/services/identity/internal/domain/recoverycodeset"
)

func toDomain(m row, rows []codeRow) *setdomain.RecoveryCodeSet {
	codes := make([]setdomain.RecoveryCode, 0, len(rows))

	for _, v := range rows {
		codes = append(
			codes,
			setdomain.RecoveryCode{
				ID:        setdomain.CodeID(v.ID),
				Hash:      setdomain.CodeHash(v.CodeHash),
				CreatedAt: v.CreatedAt,
				UsedAt:    domain.ClonePointer(v.UsedAt),
			},
		)
	}

	return &setdomain.RecoveryCodeSet{
		ID:         setdomain.ID(m.ID),
		IdentityID: identitydomain.ID(m.IdentityID),
		Codes:      codes,
		CreatedAt:  m.CreatedAt,
		ExpiresAt:  domain.ClonePointer(m.ExpiresAt),
		DisabledAt: domain.ClonePointer(m.DisabledAt),
	}
}
