package mfa

import (
	"context"
	"fmt"

	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	mfadomain "github.com/krackl1n/careeria/services/identity/internal/domain/mfa"
)

func (r *PostgresRepository) Get(
	ctx context.Context,
	id identitydomain.ID,
) (*mfadomain.MFA, error) {
	rows, err := r.client.Executor(ctx).Query(ctx, `
		SELECT
			f.id,
			f.identity_id,
			f.factor_type,
			f.name,
			f.is_primary,
			f.confirmed_at,
			f.created_at,
			f.updated_at,
			f.last_used_at,
			f.disabled_at,
			t.secret_encrypted,
			t.encryption_key_ref,
			t.algorithm,
			t.digits,
			t.period_seconds,
			t.last_used_time_step,
			t.updated_at,
			w.credential_id,
			w.public_key,
			w.sign_count,
			w.aaguid,
			w.transports,
			w.backup_eligible,
			w.backup_state,
			w.created_at,
			w.last_used_at
		FROM public.mfa_factors AS f
		LEFT JOIN public.totp_factors AS t ON t.factor_id = f.id
		LEFT JOIN public.webauthn_credentials AS w ON w.factor_id = f.id
		WHERE f.identity_id = $1
		ORDER BY f.created_at, f.id`,
		id.UUID(),
	)
	if err != nil {
		return nil, fmt.Errorf("get MFA configuration: %w", err)
	}

	defer rows.Close()

	factors, err := scanFactorRows(rows)
	if err != nil {
		return nil, err
	}

	return toDomain(id, factors), nil
}
