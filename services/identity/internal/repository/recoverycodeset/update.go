package recoverycodeset

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	setdomain "github.com/krackl1n/careeria/services/identity/internal/domain/recoverycodeset"
)

func (r *PostgresRepository) Update(ctx context.Context, e *setdomain.RecoveryCodeSet) error {
	if e == nil {
		return errors.New("update recovery code set: entity is nil")
	}

	setID := e.ID.UUID()
	b := &pgx.Batch{}

	b.Queue(`
		UPDATE public.recovery_code_sets
		SET
			expires_at = $2,
			disabled_at = $3
		WHERE id = $1`,
		setID,
		e.ExpiresAt,
		e.DisabledAt,
	).Exec(func(tag pgconn.CommandTag) error {
		if tag.RowsAffected() == 0 {
			return setdomain.ErrNotFound
		}

		return nil
	})

	for _, v := range e.Codes {
		b.Queue(`
			UPDATE public.recovery_codes
			SET used_at = $2
			WHERE id = $1
				AND set_id = $3`,
			v.ID.UUID(),
			v.UsedAt,
			setID,
		)
	}

	if err := r.client.Executor(ctx).SendBatch(ctx, b).Close(); err != nil {
		if errors.Is(err, setdomain.ErrNotFound) {
			return setdomain.ErrNotFound
		}

		return fmt.Errorf("update recovery code set: %w", err)
	}

	return nil
}
