package passwordresetchallenge

import (
	"context"
	"fmt"
	"time"

	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

func (r *PostgresRepository) RevokeActiveByIdentityID(
	ctx context.Context,
	identityID identitydomain.ID,
	revokedAt time.Time,
) (int64, error) {
	tag, err := r.client.Executor(ctx).Exec(ctx, `
		UPDATE public.password_reset_challenges
		SET revoked_at = $2
		WHERE identity_id = $1
			AND consumed_at IS NULL
			AND revoked_at IS NULL`,
		identityID.UUID(),
		revokedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("revoke active password reset challenges: %w", err)
	}

	return tag.RowsAffected(), nil
}
