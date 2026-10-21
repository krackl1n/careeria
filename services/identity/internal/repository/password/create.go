package password

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	sharedpostgres "github.com/krackl1n/careeria/postgres"
	passworddomain "github.com/krackl1n/careeria/services/identity/internal/domain/password"
)

func (r *PostgresRepository) Create(ctx context.Context, entity *passworddomain.Password) error {
	if entity == nil {
		return errors.New("create password: entity is nil")
	}

	identityID := entity.IdentityID.UUID()
	batch := &pgx.Batch{}

	batch.Queue(`
		INSERT INTO public.identity_password_credentials (
			identity_id,
			password_hash,
			must_change,
			password_updated_at,
			expires_at,
			compromised_at,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		identityID,
		string(entity.Credential.Hash),
		entity.Credential.MustChange,
		entity.Credential.PasswordUpdatedAt,
		entity.Credential.ExpiresAt,
		entity.Credential.CompromisedAt,
		entity.Credential.CreatedAt,
		entity.Credential.UpdatedAt,
	)

	batch.Queue(`
		INSERT INTO public.identity_password_auth_state (
			identity_id,
			failed_attempts,
			last_failed_at,
			locked_until,
			lock_reason,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		identityID,
		entity.AuthState.FailedAttempts,
		entity.AuthState.LastFailedAt,
		entity.AuthState.LockedUntil,
		lockReasonToRow(entity.AuthState.LockReason),
		entity.AuthState.UpdatedAt,
	)

	if err := r.client.Executor(ctx).SendBatch(ctx, batch).Close(); err != nil {
		if sharedpostgres.IsUniqueViolation(err, "identity_password_credentials_pkey") {
			return passworddomain.ErrAlreadyExists
		}

		return fmt.Errorf("create password: %w", err)
	}

	return nil
}
