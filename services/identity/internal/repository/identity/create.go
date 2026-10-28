package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

func (r *PostgresRepository) Create(ctx context.Context, entity *identitydomain.Identity) error {
	if entity == nil {
		return errors.New("create identity: entity is nil")
	}

	identityID := entity.ID.UUID()
	batch := &pgx.Batch{}

	batch.Queue(`
		INSERT INTO public.identities (
			id,
			status,
			created_at,
			updated_at,
			deleted_at
		)
		VALUES ($1, $2, $3, $4, $5)`,
		identityID,
		string(entity.Status),
		entity.CreatedAt,
		entity.UpdatedAt,
		entity.DeletedAt,
	)

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
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
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
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
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
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
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
		if conflict := conflictError(err); conflict != nil {
			return conflict
		}

		return fmt.Errorf("create identity: %w", err)
	}

	return nil
}
