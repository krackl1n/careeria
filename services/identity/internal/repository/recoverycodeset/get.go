package recoverycodeset

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	setdomain "github.com/krackl1n/careeria/services/identity/internal/domain/recoverycodeset"
)

func (r *PostgresRepository) Get(
	ctx context.Context,
	id setdomain.ID,
) (*setdomain.RecoveryCodeSet, error) {
	value := id.UUID()

	return r.get(ctx, &value, nil)
}

func (r *PostgresRepository) GetActiveByIdentityID(
	ctx context.Context,
	id identitydomain.ID,
) (*setdomain.RecoveryCodeSet, error) {
	value := id.UUID()

	return r.get(ctx, nil, &value)
}

func (r *PostgresRepository) get(
	ctx context.Context,
	id any,
	identityID any,
) (*setdomain.RecoveryCodeSet, error) {
	var m row
	var codes []codeRow
	b := &pgx.Batch{}

	b.Queue(`
		SELECT
			id,
			identity_id,
			created_at,
			expires_at,
			disabled_at
		FROM public.recovery_code_sets
		WHERE ($1::uuid IS NOT NULL AND id = $1)
			OR ($2::uuid IS NOT NULL AND identity_id = $2 AND disabled_at IS NULL)`,
		id,
		identityID,
	).QueryRow(func(row pgx.Row) error {
		v, err := scanRow(row)
		if err != nil {
			return err
		}

		m = v
		return nil
	})

	b.Queue(`
		SELECT
			c.id,
			c.set_id,
			c.code_hash,
			c.created_at,
			c.used_at
		FROM public.recovery_codes AS c
		JOIN public.recovery_code_sets AS s ON s.id = c.set_id
		WHERE ($1::uuid IS NOT NULL AND s.id = $1)
			OR ($2::uuid IS NOT NULL AND s.identity_id = $2 AND s.disabled_at IS NULL)
		ORDER BY c.created_at, c.id`,
		id,
		identityID,
	).Query(func(rows pgx.Rows) error {
		var err error
		codes, err = scanCodeRows(rows)
		if err != nil {
			return err
		}

		return nil
	})

	if err := r.client.Executor(ctx).SendBatch(ctx, b).Close(); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, setdomain.ErrNotFound
		}

		return nil, fmt.Errorf("get recovery code set: %w", err)
	}

	return toDomain(m, codes), nil
}
