package refreshtokenfamily

import (
	"context"
	"fmt"
	"time"

	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

func (r *PostgresRepository) RevokeActiveByIdentityID(
	ctx context.Context,
	id identitydomain.ID,
	reason string,
	at time.Time,
) (int64, error) {
	tag, err := r.client.Executor(ctx).Exec(ctx, `
		UPDATE public.refresh_token_families AS f
		SET revoked_at = $2, revoke_reason = $3
		FROM public.sessions AS s
		WHERE f.session_id = s.id
			AND s.identity_id = $1
			AND f.revoked_at IS NULL`,
		id.UUID(),
		at,
		reason,
	)
	if err != nil {
		return 0, fmt.Errorf("revoke identity refresh token families: %w", err)
	}

	return tag.RowsAffected(), nil
}

func (r *PostgresRepository) RevokeExpired(
	ctx context.Context,
	reason string,
	at time.Time,
) (int64, error) {
	tag, err := r.client.Executor(ctx).Exec(ctx, `
		UPDATE public.refresh_token_families
		SET revoked_at = $1, revoke_reason = $2
		WHERE revoked_at IS NULL
			AND absolute_expires_at <= $1`,
		at,
		reason,
	)
	if err != nil {
		return 0, fmt.Errorf("revoke expired refresh token families: %w", err)
	}

	return tag.RowsAffected(), nil
}
