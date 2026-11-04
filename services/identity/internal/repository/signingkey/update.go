package signingkey

import (
	"context"
	"errors"
	"fmt"

	sharedpostgres "github.com/krackl1n/careeria/postgres"
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
)

func (r *PostgresRepository) Update(
	ctx context.Context,
	entity *signingkeydomain.SigningKey,
) error {
	if entity == nil {
		return errors.New("update signing key: entity is nil")
	}

	tag, err := r.client.Executor(ctx).Exec(ctx, `
		UPDATE public.signing_keys
		SET
			status = $2,
			activated_at = $3,
			retiring_at = $4,
			retired_at = $5,
			expires_at = $6,
			compromised_at = $7
		WHERE id = $1`,
		entity.ID.UUID(),
		string(entity.Status),
		entity.ActivatedAt,
		entity.RetiringAt,
		entity.RetiredAt,
		entity.ExpiresAt,
		entity.CompromisedAt,
	)

	if err != nil {
		if sharedpostgres.IsUniqueViolation(err, "signing_keys_active_algorithm_uidx") {
			return signingkeydomain.ErrActiveAlgorithmConflict
		}

		return fmt.Errorf("update signing key: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return signingkeydomain.ErrNotFound
	}

	return nil
}
