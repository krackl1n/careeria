package signingkey

import (
	"slices"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain"
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
)

func toDomain(model row) *signingkeydomain.SigningKey {
	return &signingkeydomain.SigningKey{
		ID:            signingkeydomain.ID(model.ID),
		KeyID:         signingkeydomain.KeyID(model.KeyID),
		Algorithm:     signingkeydomain.Algorithm(model.Algorithm),
		PublicJWK:     signingkeydomain.PublicJWK(slices.Clone(model.PublicJWK)),
		PrivateKeyRef: signingkeydomain.PrivateKeyRef(model.PrivateKeyRef),
		Status:        signingkeydomain.Status(model.Status),
		CreatedAt:     model.CreatedAt,
		NotBefore:     model.NotBefore,
		ActivatedAt:   domain.ClonePointer(model.ActivatedAt),
		RetiringAt:    domain.ClonePointer(model.RetiringAt),
		RetiredAt:     domain.ClonePointer(model.RetiredAt),
		ExpiresAt:     domain.ClonePointer(model.ExpiresAt),
		CompromisedAt: domain.ClonePointer(model.CompromisedAt),
	}
}
