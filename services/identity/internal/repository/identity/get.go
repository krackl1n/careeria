package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

func (r *PostgresRepository) Get(
	ctx context.Context,
	id identitydomain.ID,
) (*identitydomain.Identity, error) {
	var identity row
	var emails []emailRow
	var phones []phoneRow
	var accounts []externalAccountRow
	batch := &pgx.Batch{}

	batch.Queue(`
		SELECT
			id,
			status,
			created_at,
			updated_at,
			deleted_at
		FROM public.identities
		WHERE id = $1
			AND deleted_at IS NULL`,
		id.UUID(),
	).QueryRow(func(row pgx.Row) error {
		scanned, err := scanRow(row)
		if err != nil {
			return err
		}

		identity = scanned
		return nil
	})

	batch.Queue(`
		SELECT
			id,
			identity_id,
			email,
			is_primary,
			verified_at,
			created_at,
			updated_at,
			deleted_at
		FROM public.identity_emails
		WHERE identity_id = $1
		ORDER BY created_at, id`,
		id.UUID(),
	).Query(func(rows pgx.Rows) error {
		var err error
		emails, err = scanEmailRows(rows)
		if err != nil {
			return err
		}

		return nil
	})

	batch.Queue(`
		SELECT
			id,
			identity_id,
			phone,
			is_primary,
			verified_at,
			created_at,
			updated_at,
			deleted_at
		FROM public.identity_phones
		WHERE identity_id = $1
		ORDER BY created_at, id`,
		id.UUID(),
	).Query(func(rows pgx.Rows) error {
		var err error
		phones, err = scanPhoneRows(rows)
		if err != nil {
			return err
		}

		return nil
	})

	batch.Queue(`
		SELECT
			id,
			identity_id,
			provider,
			subject,
			linked_at,
			last_authenticated_at,
			disabled_at,
			deleted_at
		FROM public.external_identity_accounts
		WHERE identity_id = $1
		ORDER BY linked_at, id`,
		id.UUID(),
	).Query(func(rows pgx.Rows) error {
		var err error
		accounts, err = scanExternalAccountRows(rows)
		if err != nil {
			return err
		}

		return nil
	})

	results := r.client.Executor(ctx).SendBatch(ctx, batch)
	if err := results.Close(); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, identitydomain.ErrNotFound
		}

		return nil, fmt.Errorf("get identity: %w", err)
	}

	return toDomain(identity, emails, phones, accounts), nil
}
