package session

import (
	"context"
	"fmt"
	"time"

	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	oauthclientdomain "github.com/krackl1n/careeria/services/identity/internal/domain/oauthclient"
)

func (r *PostgresRepository) RevokeActiveByIdentityID(
	ctx context.Context,
	id identitydomain.ID,
	reason string,
	revokedAt time.Time,
) (int64, error) {
	tag, err := r.client.Executor(ctx).Exec(ctx, `
		UPDATE public.sessions
		SET revoked_at = $2, revoke_reason = $3
		WHERE identity_id = $1
			AND revoked_at IS NULL`,
		id.UUID(),
		revokedAt,
		reason,
	)
	if err != nil {
		return 0, fmt.Errorf("revoke identity sessions: %w", err)
	}

	return tag.RowsAffected(), nil
}

func (r *PostgresRepository) RevokeActiveByOAuthClientID(
	ctx context.Context,
	id oauthclientdomain.ID,
	reason string,
	revokedAt time.Time,
) (int64, error) {
	tag, err := r.client.Executor(ctx).Exec(ctx, `
		UPDATE public.sessions
		SET revoked_at = $2, revoke_reason = $3
		WHERE oauth_client_id = $1
			AND revoked_at IS NULL`,
		id.UUID(),
		revokedAt,
		reason,
	)
	if err != nil {
		return 0, fmt.Errorf("revoke OAuth client sessions: %w", err)
	}

	return tag.RowsAffected(), nil
}
