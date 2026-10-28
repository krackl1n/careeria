package recoverycodeset

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	sharedpostgres "github.com/krackl1n/careeria/postgres"
	setdomain "github.com/krackl1n/careeria/services/identity/internal/domain/recoverycodeset"
)

func (r *PostgresRepository) Create(ctx context.Context, e *setdomain.RecoveryCodeSet) error {
	if e == nil {
		return errors.New("create recovery code set: entity is nil")
	}

	setID := e.ID.UUID()
	b := &pgx.Batch{}

	b.Queue(`
		INSERT INTO public.recovery_code_sets (
			id,
			identity_id,
			created_at,
			expires_at,
			disabled_at
		)
		VALUES ($1, $2, $3, $4, $5)`,
		setID,
		e.IdentityID.UUID(),
		e.CreatedAt,
		e.ExpiresAt,
		e.DisabledAt,
	)

	for _, v := range e.Codes {
		b.Queue(`
			INSERT INTO public.recovery_codes (
				id,
				set_id,
				code_hash,
				created_at,
				used_at
			)
			VALUES ($1, $2, $3, $4, $5)`,
			v.ID.UUID(),
			setID,
			string(v.Hash),
			v.CreatedAt,
			v.UsedAt,
		)
	}

	if err := r.client.Executor(ctx).SendBatch(ctx, b).Close(); err != nil {
		if sharedpostgres.IsUniqueViolation(err, "recovery_code_sets_active_uidx") {
			return setdomain.ErrActiveSetAlreadyExists
		}

		return fmt.Errorf("create recovery code set: %w", err)
	}

	return nil
}
