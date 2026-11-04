package refreshtokenfamily

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	domain "github.com/krackl1n/careeria/services/identity/internal/domain/refreshtokenfamily"
)

func (r *PostgresRepository) Get(
	ctx context.Context,
	id domain.ID,
) (*domain.RefreshTokenFamily, error) {
	return r.get(ctx, id.UUID(), false)
}

func (r *PostgresRepository) GetByTokenHash(
	ctx context.Context,
	hash domain.TokenHash,
) (*domain.RefreshTokenFamily, error) {
	var id [16]byte
	err := r.client.Executor(ctx).QueryRow(ctx, `
		SELECT family_id
		FROM public.refresh_tokens
		WHERE token_hash = $1`,
		string(hash),
	).Scan(&id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTokenNotFound
		}

		return nil, fmt.Errorf("find refresh token family: %w", err)
	}

	return r.get(ctx, id, false)
}

func (r *PostgresRepository) GetByTokenHashForUpdate(
	ctx context.Context,
	hash domain.TokenHash,
) (*domain.RefreshTokenFamily, error) {
	var id [16]byte
	err := r.client.Executor(ctx).QueryRow(ctx, `
		SELECT family_id
		FROM public.refresh_tokens
		WHERE token_hash = $1`,
		string(hash),
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTokenNotFound
		}

		return nil, fmt.Errorf("find refresh token family for update: %w", err)
	}

	return r.get(ctx, id, true)
}

func (r *PostgresRepository) get(
	ctx context.Context,
	id any,
	forUpdate bool,
) (*domain.RefreshTokenFamily, error) {
	var m row
	var tokens []tokenRow
	b := &pgx.Batch{}

	familyQuery := `
		SELECT
			id,
			session_id,
			created_at,
			last_rotated_at,
			absolute_expires_at,
			reuse_detected_at,
			revoked_at,
			revoke_reason
		FROM public.refresh_token_families
		WHERE id = $1`
	if forUpdate {
		familyQuery += " FOR UPDATE"
	}
	b.Queue(familyQuery,
		id,
	).QueryRow(func(row pgx.Row) error {
		scanned, err := scanRow(row)
		if err != nil {
			return err
		}

		m = scanned
		return nil
	})

	b.Queue(`
		SELECT
			id,
			family_id,
			parent_token_id,
			token_hash,
			issued_at,
			expires_at,
			rotated_at,
			revoked_at,
			revoke_reason,
			reuse_detected_at,
			created_from_ip,
			created_user_agent
		FROM public.refresh_tokens
		WHERE family_id = $1
		ORDER BY issued_at, id`,
		id,
	).Query(func(rows pgx.Rows) error {
		var err error
		tokens, err = scanTokenRows(rows)
		if err != nil {
			return err
		}

		return nil
	})

	if err := r.client.Executor(ctx).SendBatch(ctx, b).Close(); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}

		return nil, fmt.Errorf("get refresh token family: %w", err)
	}

	return toDomain(m, tokens), nil
}
