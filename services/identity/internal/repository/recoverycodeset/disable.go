package recoverycodeset

import (
	"context"
	"fmt"
	"time"

	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
)

func (r *PostgresRepository) DisableActiveByIdentityID(
	ctx context.Context,
	id identitydomain.ID,
	at time.Time,
) (int64, error) {
	tag, err := r.client.Executor(ctx).Exec(ctx, `
		UPDATE public.recovery_code_sets
		SET disabled_at = $2
		WHERE identity_id = $1
			AND disabled_at IS NULL`,
		id.UUID(),
		at,
	)

	if err != nil {
		return 0, fmt.Errorf("disable active recovery code sets: %w", err)
	}

	return tag.RowsAffected(), nil
}
