package signingkey

import (
	"context"
	"errors"
	"fmt"

	sharedpostgres "github.com/krackl1n/careeria/postgres"
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
)

func (r *PostgresRepository) Create(
	ctx context.Context,
	entity *signingkeydomain.SigningKey,
) error {
	if entity == nil {
		return errors.New("create signing key: entity is nil")
	}

	_, err := r.client.Executor(ctx).Exec(ctx, `
		INSERT INTO public.signing_keys (
			id,
			kid,
			algorithm,
			public_jwk,
			private_key_ref,
			status,
			created_at,
			not_before,
			activated_at,
			retiring_at,
			retired_at,
			expires_at,
			compromised_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		entity.ID.UUID(),
		string(entity.KeyID),
		string(entity.Algorithm),
		[]byte(entity.PublicJWK),
		string(entity.PrivateKeyRef),
		string(entity.Status),
		entity.CreatedAt,
		entity.NotBefore,
		entity.ActivatedAt,
		entity.RetiringAt,
		entity.RetiredAt,
		entity.ExpiresAt,
		entity.CompromisedAt,
	)

	if err != nil {
		if sharedpostgres.IsUniqueViolation(err, "signing_keys_kid_uidx") {
			return signingkeydomain.ErrKeyIDAlreadyExists
		}
		if sharedpostgres.IsUniqueViolation(err, "signing_keys_active_algorithm_uidx") {
			return signingkeydomain.ErrActiveAlgorithmConflict
		}

		return fmt.Errorf("create signing key: %w", err)
	}

	return nil
}
