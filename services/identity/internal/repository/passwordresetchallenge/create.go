package passwordresetchallenge

import (
	"context"
	"errors"
	"fmt"

	challenge "github.com/krackl1n/careeria/services/identity/internal/domain/passwordresetchallenge"
)

func (r *PostgresRepository) Create(
	ctx context.Context,
	entity *challenge.PasswordResetChallenge,
) error {
	if entity == nil {
		return errors.New("create password reset challenge: entity is nil")
	}

	_, err := r.client.Executor(ctx).Exec(ctx, `
		INSERT INTO public.password_reset_challenges (
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
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		entity.ID.UUID(),
		entity.IdentityID.UUID(),
		string(entity.TokenHash),
		entity.Attempts,
		entity.MaxAttempts,
		ipAddressToRow(entity.RequestedFromIP),
		ipAddressToRow(entity.ConsumedFromIP),
		entity.CreatedAt,
		entity.ExpiresAt,
		entity.LastAttemptAt,
		entity.ConsumedAt,
		entity.RevokedAt,
	)
	if err != nil {
		return fmt.Errorf("create password reset challenge: %w", err)
	}

	return nil
}
