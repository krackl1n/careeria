package password

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	passworddomain "github.com/krackl1n/careeria/services/identity/internal/domain/password"
)

func (r *PostgresRepository) Update(ctx context.Context, entity *passworddomain.Password) error {
	if entity == nil {
		return errors.New("update password: entity is nil")
	}

	identityID := entity.IdentityID.UUID()
	batch := &pgx.Batch{}

	batch.Queue(`
		UPDATE public.identity_password_credentials
		SET
			password_hash = $2,
			must_change = $3,
			password_updated_at = $4,
			expires_at = $5,
			compromised_at = $6,
			updated_at = $7
		WHERE identity_id = $1`,
		identityID,
		string(entity.Credential.Hash),
		entity.Credential.MustChange,
		entity.Credential.PasswordUpdatedAt,
		entity.Credential.ExpiresAt,
		entity.Credential.CompromisedAt,
		entity.Credential.UpdatedAt,
	).Exec(func(tag pgconn.CommandTag) error {
		if tag.RowsAffected() == 0 {
			return passworddomain.ErrNotFound
		}

		return nil
	})

	batch.Queue(`
		UPDATE public.identity_password_auth_state
		SET
			failed_attempts = $2,
			last_failed_at = $3,
			locked_until = $4,
			lock_reason = $5,
			updated_at = $6
		WHERE identity_id = $1`,
		identityID,
		entity.AuthState.FailedAttempts,
		entity.AuthState.LastFailedAt,
		entity.AuthState.LockedUntil,
		lockReasonToRow(entity.AuthState.LockReason),
		entity.AuthState.UpdatedAt,
	)

	if err := r.client.Executor(ctx).SendBatch(ctx, batch).Close(); err != nil {
		if errors.Is(err, passworddomain.ErrNotFound) {
			return passworddomain.ErrNotFound
		}

		return fmt.Errorf("update password: %w", err)
	}

	return nil
}
