package serviceaccount

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	sharedpostgres "github.com/krackl1n/careeria/postgres"
	accountdomain "github.com/krackl1n/careeria/services/identity/internal/domain/serviceaccount"
)

func (r *PostgresRepository) Create(ctx context.Context, e *accountdomain.ServiceAccount) error {
	if e == nil {
		return errors.New("create service account: entity is nil")
	}

	b := &pgx.Batch{}

	b.Queue(`
		INSERT INTO public.service_accounts (
			id,
			organization_id,
			account_key,
			name,
			description,
			status,
			created_by_identity_id,
			created_at,
			updated_at,
			disabled_at,
			deleted_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		e.ID.UUID(),
		e.OrganizationID.UUID(),
		string(e.AccountKey),
		e.Name,
		e.Description,
		string(e.Status),
		identityIDToRow(e.CreatedByIdentityID),
		e.CreatedAt,
		e.UpdatedAt,
		e.DisabledAt,
		e.DeletedAt,
	)

	queueCredentials(b, e)

	return r.finishSave(ctx, b, "create")
}

func (r *PostgresRepository) Update(ctx context.Context, e *accountdomain.ServiceAccount) error {
	if e == nil {
		return errors.New("update service account: entity is nil")
	}

	b := &pgx.Batch{}

	b.Queue(`
		UPDATE public.service_accounts
		SET
			name = $2,
			description = $3,
			status = $4,
			updated_at = $5,
			disabled_at = $6,
			deleted_at = $7
		WHERE id = $1`,
		e.ID.UUID(),
		e.Name,
		e.Description,
		string(e.Status),
		e.UpdatedAt,
		e.DisabledAt,
		e.DeletedAt,
	).Exec(func(tag pgconn.CommandTag) error {
		if tag.RowsAffected() == 0 {
			return accountdomain.ErrNotFound
		}

		return nil
	})

	queueCredentials(b, e)

	return r.finishSave(ctx, b, "update")
}

func queueCredentials(b *pgx.Batch, e *accountdomain.ServiceAccount) {
	for _, credential := range e.Credentials {
		v := credentialToRow(e.ID, credential)
		b.Queue(`
			INSERT INTO public.service_account_credentials (
				id,
				service_account_id,
				credential_type,
				secret_hash,
				secret_prefix,
				name,
				created_by_identity_id,
				created_at,
				updated_at,
				expires_at,
				revoked_at,
				revoke_reason,
				last_used_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT (id) DO UPDATE
			SET
				secret_prefix = EXCLUDED.secret_prefix,
				name = EXCLUDED.name,
				updated_at = EXCLUDED.updated_at,
				expires_at = EXCLUDED.expires_at,
				revoked_at = EXCLUDED.revoked_at,
				revoke_reason = EXCLUDED.revoke_reason,
				last_used_at = EXCLUDED.last_used_at`,
			v.ID,
			v.ServiceAccountID,
			v.Type,
			v.SecretHash,
			v.SecretPrefix,
			v.Name,
			v.CreatedByIdentityID,
			v.CreatedAt,
			v.UpdatedAt,
			v.ExpiresAt,
			v.RevokedAt,
			v.RevokeReason,
			v.LastUsedAt,
		)
	}
}

func (r *PostgresRepository) finishSave(ctx context.Context, b *pgx.Batch, op string) error {
	if err := r.client.Executor(ctx).SendBatch(ctx, b).Close(); err != nil {
		if errors.Is(err, accountdomain.ErrNotFound) {
			return accountdomain.ErrNotFound
		}

		if sharedpostgres.IsUniqueViolation(err, "service_accounts_org_key_uidx") {
			return accountdomain.ErrAccountKeyAlreadyExists
		}

		return fmt.Errorf("%s service account: %w", op, err)
	}

	return nil
}
