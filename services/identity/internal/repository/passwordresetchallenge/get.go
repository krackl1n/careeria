package passwordresetchallenge

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	challenge "github.com/krackl1n/careeria/services/identity/internal/domain/passwordresetchallenge"
)

func (r *PostgresRepository) Get(
	ctx context.Context,
	id challenge.ID,
) (*challenge.PasswordResetChallenge, error) {
	model, err := scanRow(r.client.Executor(ctx).QueryRow(ctx, `
		SELECT
			id,
			identity_id,
			token_hash,
			attempts,
			max_attempts,
			requested_from_ip,
			consumed_from_ip,
			created_at,
			expires_at,
			last_attempt_at,
			consumed_at,
			revoked_at
		FROM public.password_reset_challenges
		WHERE id = $1`,
		id.UUID(),
	))

	return finishGet(model, err)
}

func (r *PostgresRepository) GetByTokenHash(
	ctx context.Context,
	hash challenge.TokenHash,
) (*challenge.PasswordResetChallenge, error) {
	model, err := scanRow(r.client.Executor(ctx).QueryRow(ctx, `
		SELECT
			id,
			identity_id,
			token_hash,
			attempts,
			max_attempts,
			requested_from_ip,
			consumed_from_ip,
			created_at,
			expires_at,
			last_attempt_at,
			consumed_at,
			revoked_at
		FROM public.password_reset_challenges
		WHERE token_hash = $1`,
		string(hash),
	))

	return finishGet(model, err)
}

func finishGet(model row, err error) (*challenge.PasswordResetChallenge, error) {
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, challenge.ErrNotFound
		}

		return nil, fmt.Errorf("get password reset challenge: %w", err)
	}

	return toDomain(model), nil
}
