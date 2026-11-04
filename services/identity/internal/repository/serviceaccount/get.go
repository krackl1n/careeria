package serviceaccount

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	accountdomain "github.com/krackl1n/careeria/services/identity/internal/domain/serviceaccount"
)

func (r *PostgresRepository) Get(
	ctx context.Context,
	id accountdomain.ID,
) (*accountdomain.ServiceAccount, error) {
	value := id.UUID()
	return r.get(ctx, &value, nil, nil)
}

func (r *PostgresRepository) GetByOrganizationAndKey(
	ctx context.Context,
	organizationID accountdomain.OrganizationID,
	key accountdomain.AccountKey,
) (*accountdomain.ServiceAccount, error) {
	organizationValue := organizationID.UUID()
	keyValue := string(key)

	return r.get(ctx, nil, &organizationValue, &keyValue)
}

func (r *PostgresRepository) get(
	ctx context.Context,
	id any,
	organizationID any,
	key *string,
) (*accountdomain.ServiceAccount, error) {
	var m row
	var credentials []credentialRow
	b := &pgx.Batch{}

	b.Queue(`
		SELECT
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
		FROM public.service_accounts
		WHERE deleted_at IS NULL
			AND (
				($1::uuid IS NOT NULL AND id = $1)
				OR ($2::uuid IS NOT NULL AND organization_id = $2 AND account_key = $3)
			)`,
		id,
		organizationID,
		key,
	).QueryRow(func(row pgx.Row) error {
		v, err := scanRow(row)
		if err != nil {
			return err
		}

		m = v
		return nil
	})

	b.Queue(`
		SELECT
			c.id,
			c.service_account_id,
			c.credential_type,
			c.secret_hash,
			c.secret_prefix,
			c.name,
			c.created_by_identity_id,
			c.created_at,
			c.updated_at,
			c.expires_at,
			c.revoked_at,
			c.revoke_reason,
			c.last_used_at
		FROM public.service_account_credentials AS c
		JOIN public.service_accounts AS a ON a.id = c.service_account_id
		WHERE a.deleted_at IS NULL
			AND (
				($1::uuid IS NOT NULL AND a.id = $1)
				OR ($2::uuid IS NOT NULL AND a.organization_id = $2 AND a.account_key = $3)
			)
		ORDER BY c.created_at, c.id`,
		id,
		organizationID,
		key,
	).Query(func(rows pgx.Rows) error {
		var err error
		credentials, err = scanCredentialRows(rows)
		if err != nil {
			return err
		}

		return nil
	})

	if err := r.client.Executor(ctx).SendBatch(ctx, b).Close(); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, accountdomain.ErrNotFound
		}

		return nil, fmt.Errorf("get service account: %w", err)
	}

	return toDomain(m, credentials), nil
}
