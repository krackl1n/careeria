package emailverificationchallenge

import (
	"context"
	"fmt"
	"time"

	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

func (r *PostgresRepository) RevokeActiveByEmailID(
	ctx context.Context,
	emailID identitydomain.EmailID,
	revokedAt time.Time,
) (int64, error) {
	tag, err := r.client.Executor(ctx).Exec(ctx, `
		UPDATE public.email_verification_challenges
		SET revoked_at = $2
		WHERE email_id = $1
			AND consumed_at IS NULL
			AND revoked_at IS NULL`,
		emailID.UUID(),
		revokedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("revoke active email verification challenges: %w", err)
	}

	return tag.RowsAffected(), nil
}
