package refreshtokenfamily

import (
	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	refreshtokenfamilydomain "github.com/krackl1n/careeria/services/identity/internal/domain/refreshtokenfamily"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

func toDomain(m row, rows []tokenRow) *refreshtokenfamilydomain.RefreshTokenFamily {
	tokens := make([]refreshtokenfamilydomain.RefreshToken, 0, len(rows))

	for _, raw := range rows {
		var parent *refreshtokenfamilydomain.TokenID
		if raw.ParentTokenID != nil {
			v := refreshtokenfamilydomain.TokenID(*raw.ParentTokenID)
			parent = &v
		}

		tokens = append(
			tokens,
			refreshtokenfamilydomain.RefreshToken{
				ID:               refreshtokenfamilydomain.TokenID(raw.ID),
				ParentTokenID:    parent,
				Hash:             refreshtokenfamilydomain.TokenHash(raw.TokenHash),
				IssuedAt:         raw.IssuedAt,
				ExpiresAt:        raw.ExpiresAt,
				RotatedAt:        domain.ClonePointer(raw.RotatedAt),
				RevokedAt:        domain.ClonePointer(raw.RevokedAt),
				RevokeReason:     domain.ClonePointer(raw.RevokeReason),
				ReuseDetectedAt:  domain.ClonePointer(raw.ReuseDetectedAt),
				CreatedFromIP:    ipToDomain(raw.CreatedFromIP),
				CreatedUserAgent: domain.ClonePointer(raw.CreatedUserAgent),
			},
		)
	}

	return &refreshtokenfamilydomain.RefreshTokenFamily{
		ID:                refreshtokenfamilydomain.ID(m.ID),
		SessionID:         sessiondomain.ID(m.SessionID),
		Tokens:            tokens,
		CreatedAt:         m.CreatedAt,
		LastRotatedAt:     domain.ClonePointer(m.LastRotatedAt),
		AbsoluteExpiresAt: m.AbsoluteExpiresAt,
		ReuseDetectedAt:   domain.ClonePointer(m.ReuseDetectedAt),
		RevokedAt:         domain.ClonePointer(m.RevokedAt),
		RevokeReason:      domain.ClonePointer(m.RevokeReason),
	}
}

func tokenToRow(
	familyID refreshtokenfamilydomain.ID,
	t refreshtokenfamilydomain.RefreshToken,
) tokenRow {
	result := tokenRow{
		ID:               t.ID.UUID(),
		FamilyID:         familyID.UUID(),
		TokenHash:        string(t.Hash),
		IssuedAt:         t.IssuedAt,
		ExpiresAt:        t.ExpiresAt,
		RotatedAt:        domain.ClonePointer(t.RotatedAt),
		RevokedAt:        domain.ClonePointer(t.RevokedAt),
		RevokeReason:     domain.ClonePointer(t.RevokeReason),
		ReuseDetectedAt:  domain.ClonePointer(t.ReuseDetectedAt),
		CreatedFromIP:    ipToRow(t.CreatedFromIP),
		CreatedUserAgent: domain.ClonePointer(t.CreatedUserAgent),
	}

	if t.ParentTokenID != nil {
		v := t.ParentTokenID.UUID()
		result.ParentTokenID = &v
	}

	return result
}

func ipToDomain(v *string) *sessiondomain.IPAddress {
	if v == nil {
		return nil
	}

	a := sessiondomain.IPAddress(*v)
	return &a
}

func ipToRow(v *sessiondomain.IPAddress) *string {
	if v == nil {
		return nil
	}

	a := string(*v)
	return &a
}
