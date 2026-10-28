package password

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	passworddomain "github.com/krackl1n/careeria/services/identity/internal/domain/password"
)

func (r *PostgresRepository) Get(
	ctx context.Context,
	identityID identitydomain.ID,
) (*passworddomain.Password, error) {
	var credential credentialRow
	var authState authStateRow
	batch := &pgx.Batch{}

	batch.Queue(`
		SELECT
			identity_id,
			password_hash,
			must_change,
			password_updated_at,
			expires_at,
			compromised_at,
			created_at,
			updated_at
		FROM public.identity_password_credentials
		WHERE identity_id = $1`,
		identityID.UUID(),
	).QueryRow(func(row pgx.Row) error {
		scanned, err := scanCredentialRow(row)
		if err != nil {
			return err
		}

		credential = scanned
		return nil
	})

	batch.Queue(`
		SELECT
			identity_id,
			failed_attempts,
			last_failed_at,
			locked_until,
			lock_reason,
			updated_at
		FROM public.identity_password_auth_state
		WHERE identity_id = $1`,
		identityID.UUID(),
	).QueryRow(func(row pgx.Row) error {
		var err error
		authState, err = scanAuthStateRow(row)
		return err
	})

	if err := r.client.Executor(ctx).SendBatch(ctx, batch).Close(); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, passworddomain.ErrNotFound
		}

		return nil, fmt.Errorf("get password: %w", err)
	}

	return toDomain(credential, authState), nil
}
