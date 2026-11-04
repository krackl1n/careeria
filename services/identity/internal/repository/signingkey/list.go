package signingkey

import (
	"context"
	"fmt"
	"time"

	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
)

func (r *PostgresRepository) ListPublic(
	ctx context.Context,
	now time.Time,
) ([]*signingkeydomain.SigningKey, error) {
	rows, err := r.client.Executor(ctx).Query(ctx, `
		SELECT
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
		FROM public.signing_keys
		WHERE status IN ('active', 'retiring')
			AND not_before <= $1
			AND (expires_at IS NULL OR expires_at > $1)
		ORDER BY created_at, id`,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("list public signing keys: %w", err)
	}
	defer rows.Close()

	models, err := scanRows(rows)
	if err != nil {
		return nil, err
	}

	entities := make([]*signingkeydomain.SigningKey, 0, len(models))
	for _, model := range models {
		entities = append(entities, toDomain(model))
	}

	return entities, nil
}
