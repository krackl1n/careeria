package passwordresetchallenge

import (
	"context"
	"errors"
	"fmt"

	challenge "github.com/krackl1n/careeria/services/identity/internal/domain/passwordresetchallenge"
)

func (r *PostgresRepository) Update(
	ctx context.Context,
	entity *challenge.PasswordResetChallenge,
) error {
	if entity == nil {
		return errors.New("update password reset challenge: entity is nil")
	}

	tag, err := r.client.Executor(ctx).Exec(ctx, `
		UPDATE public.password_reset_challenges
		SET
			attempts = $2,
			consumed_from_ip = $3,
			last_attempt_at = $4,
			consumed_at = $5,
			revoked_at = $6
		WHERE id = $1`,
		entity.ID.UUID(),
		entity.Attempts,
		ipAddressToRow(entity.ConsumedFromIP),
		entity.LastAttemptAt,
		entity.ConsumedAt,
		entity.RevokedAt,
	)
	if err != nil {
		return fmt.Errorf("update password reset challenge: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return challenge.ErrNotFound
	}

	return nil
}
