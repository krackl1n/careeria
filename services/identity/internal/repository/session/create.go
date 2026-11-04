package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

func (r *PostgresRepository) Create(ctx context.Context, entity *sessiondomain.Session) error {
	if entity == nil {
		return errors.New("create session: entity is nil")
	}

	model := toRow(entity)
	batch := &pgx.Batch{}

	batch.Queue(`
		INSERT INTO public.sessions (
			id,
			identity_id,
			oauth_client_id,
			authentication_level,
			authenticated_methods,
			ip_address,
			user_agent,
			device_name,
			created_at,
			authenticated_at,
			last_seen_at,
			idle_expires_at,
			absolute_expires_at,
			revoked_at,
			revoke_reason
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		model.ID,
		model.IdentityID,
		model.OAuthClientID,
		model.AuthenticationLevel,
		model.AuthenticatedMethods,
		model.IPAddress,
		model.UserAgent,
		model.DeviceName,
		model.CreatedAt,
		model.AuthenticatedAt,
		model.LastSeenAt,
		model.IdleExpiresAt,
		model.AbsoluteExpiresAt,
		model.RevokedAt,
		model.RevokeReason,
	)

	for _, scope := range model.Scopes {
		batch.Queue(`
			INSERT INTO public.session_scopes (
				session_id,
				scope
			)
			VALUES ($1, $2)`,
			model.ID,
			scope,
		)
	}

	if err := r.client.Executor(ctx).SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}
