package signingkey

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	signingkeydomain "github.com/krackl1n/careeria/services/identity/internal/domain/signingkey"
)

func (r *PostgresRepository) GetActive(
	ctx context.Context,
	algorithm signingkeydomain.Algorithm,
	now time.Time,
) (*signingkeydomain.SigningKey, error) {
	model, err := scanRow(r.client.Executor(ctx).QueryRow(ctx, `
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
		WHERE algorithm = $1
			AND status = 'active'
			AND not_before <= $2
			AND (expires_at IS NULL OR expires_at > $2)`,
		algorithm,
		now,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, signingkeydomain.ErrNotFound
		}

		return nil, fmt.Errorf("get active signing key: %w", err)
	}

	return toDomain(model), nil
}

func (r *PostgresRepository) Get(
	ctx context.Context,
	id signingkeydomain.ID,
) (*signingkeydomain.SigningKey, error) {
	value := id.UUID()

	return r.get(ctx, &value, nil)
}

func (r *PostgresRepository) GetByKeyID(
	ctx context.Context,
	keyID signingkeydomain.KeyID,
) (*signingkeydomain.SigningKey, error) {
	value := string(keyID)

	return r.get(ctx, nil, &value)
}

func (r *PostgresRepository) get(
	ctx context.Context,
	id any,
	keyID *string,
) (*signingkeydomain.SigningKey, error) {
	model, err := scanRow(r.client.Executor(ctx).QueryRow(ctx, `
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
		WHERE ($1::uuid IS NOT NULL AND id = $1)
			OR ($2::text IS NOT NULL AND kid = $2)`,
		id,
		keyID,
	))

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, signingkeydomain.ErrNotFound
		}

		return nil, fmt.Errorf("get signing key: %w", err)
	}

	return toDomain(model), nil
}
