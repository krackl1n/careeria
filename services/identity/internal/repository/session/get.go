package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

func (r *PostgresRepository) Get(
	ctx context.Context,
	id sessiondomain.ID,
) (*sessiondomain.Session, error) {
	model, err := scanRow(r.client.Executor(ctx).QueryRow(ctx, `
		SELECT
			s.id,
			s.identity_id,
			s.oauth_client_id,
			s.authentication_level,
			s.authenticated_methods,
			COALESCE(
				array_agg(ss.scope ORDER BY ss.scope) FILTER (WHERE ss.scope IS NOT NULL),
				ARRAY[]::text[]
			),
			s.ip_address,
			s.user_agent,
			s.device_name,
			s.created_at,
			s.authenticated_at,
			s.last_seen_at,
			s.idle_expires_at,
			s.absolute_expires_at,
			s.revoked_at,
			s.revoke_reason
		FROM public.sessions AS s
		LEFT JOIN public.session_scopes AS ss ON ss.session_id = s.id
		WHERE s.id = $1
		GROUP BY s.id`,
		id.UUID(),
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, sessiondomain.ErrNotFound
		}

		return nil, fmt.Errorf("get session: %w", err)
	}

	return toDomain(model), nil
}
