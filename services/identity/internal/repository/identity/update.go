package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

func (r *PostgresRepository) Update(ctx context.Context, entity *identitydomain.Identity) error {
	if entity == nil {
		return errors.New("update identity: entity is nil")
	}

	if entity.DeletedAt != nil || entity.Status == identitydomain.StatusDeleted {
		return errors.New("update identity: active identity is required")
	}

	identityID := entity.ID.UUID()
	batch := &pgx.Batch{}

	batch.Queue(`
		UPDATE public.identities
		SET
			status = $2,
			updated_at = $3
		WHERE id = $1
			AND deleted_at IS NULL`,
		identityID,
		string(entity.Status),
		entity.UpdatedAt,
	).Exec(func(tag pgconn.CommandTag) error {
		if tag.RowsAffected() == 0 {
			return identitydomain.ErrNotFound
		}

		return nil
	})

	for _, email := range entity.Emails {
		batch.Queue(`
			INSERT INTO public.identity_emails (
				id,
				identity_id,
				email,
				is_primary,
				verified_at,
				created_at,
				updated_at,
				deleted_at
			)
			SELECT $1, $2, $3, $4, $5, $6, $7, $8
			WHERE EXISTS (
				SELECT 1
				FROM public.identities
				WHERE id = $2
					AND deleted_at IS NULL
			)
			ON CONFLICT (id) DO UPDATE
			SET
				identity_id = EXCLUDED.identity_id,
				email = EXCLUDED.email,
				is_primary = EXCLUDED.is_primary,
				verified_at = EXCLUDED.verified_at,
				updated_at = EXCLUDED.updated_at,
				deleted_at = EXCLUDED.deleted_at`,
			email.ID.UUID(),
			identityID,
			string(email.Address),
			email.IsPrimary,
			email.VerifiedAt,
			email.CreatedAt,
			email.UpdatedAt,
			email.DeletedAt,
		)
	}

	for _, phone := range entity.Phones {
		batch.Queue(`
			INSERT INTO public.identity_phones (
				id,
				identity_id,
				phone,
				is_primary,
				verified_at,
				created_at,
				updated_at,
				deleted_at
			)
			SELECT $1, $2, $3, $4, $5, $6, $7, $8
			WHERE EXISTS (
				SELECT 1
				FROM public.identities
				WHERE id = $2
					AND deleted_at IS NULL
			)
			ON CONFLICT (id) DO UPDATE
			SET
				identity_id = EXCLUDED.identity_id,
				phone = EXCLUDED.phone,
				is_primary = EXCLUDED.is_primary,
				verified_at = EXCLUDED.verified_at,
				updated_at = EXCLUDED.updated_at,
				deleted_at = EXCLUDED.deleted_at`,
			phone.ID.UUID(),
			identityID,
			string(phone.Number),
			phone.IsPrimary,
			phone.VerifiedAt,
			phone.CreatedAt,
			phone.UpdatedAt,
			phone.DeletedAt,
		)
	}

	for _, account := range entity.ExternalAccounts {
		batch.Queue(`
			INSERT INTO public.external_identity_accounts (
				id,
				identity_id,
				provider,
				subject,
				linked_at,
				last_authenticated_at,
				disabled_at,
				deleted_at
			)
			SELECT $1, $2, $3, $4, $5, $6, $7, $8
			WHERE EXISTS (
				SELECT 1
				FROM public.identities
				WHERE id = $2
					AND deleted_at IS NULL
			)
			ON CONFLICT (id) DO UPDATE
			SET
				identity_id = EXCLUDED.identity_id,
				provider = EXCLUDED.provider,
				subject = EXCLUDED.subject,
				linked_at = EXCLUDED.linked_at,
				last_authenticated_at = EXCLUDED.last_authenticated_at,
				disabled_at = EXCLUDED.disabled_at,
				deleted_at = EXCLUDED.deleted_at`,
			account.ID.UUID(),
			identityID,
			string(account.Provider),
			string(account.Subject),
			account.LinkedAt,
			account.LastAuthenticatedAt,
			account.DisabledAt,
			account.DeletedAt,
		)
	}

	if err := r.client.Executor(ctx).SendBatch(ctx, batch).Close(); err != nil {
		if errors.Is(err, identitydomain.ErrNotFound) {
			return identitydomain.ErrNotFound
		}
		if conflict := conflictError(err); conflict != nil {
			return conflict
		}

		return fmt.Errorf("update identity: %w", err)
	}

	return nil
}
