package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

func (r *PostgresRepository) Update(ctx context.Context, entity *sessiondomain.Session) error {
	if entity == nil {
		return errors.New("update session: entity is nil")
	}

	model := toRow(entity)
	batch := &pgx.Batch{}

	batch.Queue(`
		UPDATE public.sessions
		SET
			authentication_level = $2,
			authenticated_methods = $3,
			ip_address = $4,
			user_agent = $5,
			device_name = $6,
			last_seen_at = $7,
			idle_expires_at = $8,
			absolute_expires_at = $9,
			revoked_at = $10,
			revoke_reason = $11
		WHERE id = $1`,
		model.ID,
		model.AuthenticationLevel,
		model.AuthenticatedMethods,
		model.IPAddress,
		model.UserAgent,
		model.DeviceName,
		model.LastSeenAt,
		model.IdleExpiresAt,
		model.AbsoluteExpiresAt,
		model.RevokedAt,
		model.RevokeReason,
	).Exec(func(tag pgconn.CommandTag) error {
		if tag.RowsAffected() == 0 {
			return sessiondomain.ErrNotFound
		}

		return nil
	})

	batch.Queue(`
		DELETE FROM public.session_scopes
		WHERE session_id = $1`,
		model.ID,
	)

	for _, scope := range model.Scopes {
		batch.Queue(`
			INSERT INTO public.session_scopes (
				session_id,
				scope
			)
			SELECT $1, $2
			WHERE EXISTS (
				SELECT 1
				FROM public.sessions
				WHERE id = $1
			)`,
			model.ID,
			scope,
		)
	}

	if err := r.client.Executor(ctx).SendBatch(ctx, batch).Close(); err != nil {
		if errors.Is(err, sessiondomain.ErrNotFound) {
			return sessiondomain.ErrNotFound
		}

		return fmt.Errorf("update session: %w", err)
	}

	return nil
}
