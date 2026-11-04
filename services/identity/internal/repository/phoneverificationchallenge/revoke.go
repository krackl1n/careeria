package phoneverificationchallenge

import (
	"context"
	"fmt"
	"time"

	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

func (r *PostgresRepository) RevokeActiveByPhoneID(
	ctx context.Context,
	phoneID identitydomain.PhoneID,
	revokedAt time.Time,
) (int64, error) {
	tag, err := r.client.Executor(ctx).Exec(ctx, `
		UPDATE public.phone_verification_challenges
		SET revoked_at = $2
		WHERE phone_id = $1
			AND consumed_at IS NULL
			AND revoked_at IS NULL`,
		phoneID.UUID(),
		revokedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("revoke active phone verification challenges: %w", err)
	}

	return tag.RowsAffected(), nil
}
