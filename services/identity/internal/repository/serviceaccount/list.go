package serviceaccount

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	accountdomain "github.com/krackl1n/careeria/services/identity/internal/domain/serviceaccount"
)

func (r *PostgresRepository) ListByOrganizationID(
	ctx context.Context,
	id accountdomain.OrganizationID,
) ([]*accountdomain.ServiceAccount, error) {
	rows, err := r.client.Executor(ctx).Query(ctx, `
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
		WHERE organization_id = $1
			AND deleted_at IS NULL
		ORDER BY created_at, id`,
		id.UUID(),
	)
	if err != nil {
		return nil, fmt.Errorf("list service accounts: %w", err)
	}

	models, err := scanRows(rows)
	rows.Close()
	if err != nil {
		return nil, err
	}

	credentialRows, err := r.client.Executor(ctx).Query(ctx, `
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
		WHERE a.organization_id = $1
			AND a.deleted_at IS NULL
		ORDER BY c.created_at, c.id`,
		id.UUID(),
	)
	if err != nil {
		return nil, fmt.Errorf("list service account credentials: %w", err)
	}

	credentials, err := scanCredentialRows(credentialRows)
	credentialRows.Close()
	if err != nil {
		return nil, err
	}

	byAccountID := make(map[uuid.UUID][]credentialRow, len(models))
	for _, v := range credentials {
		byAccountID[v.ServiceAccountID] = append(byAccountID[v.ServiceAccountID], v)
	}

	entities := make([]*accountdomain.ServiceAccount, 0, len(models))
	for _, m := range models {
		entities = append(entities, toDomain(m, byAccountID[m.ID]))
	}

	return entities, nil
}
