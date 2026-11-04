package session

import (
	"context"
	"fmt"

	identitydomain "github.com/krackl1n/careeria/services/identity/internal/domain/identity"
	sessiondomain "github.com/krackl1n/careeria/services/identity/internal/domain/session"
)

func (r *PostgresRepository) ListByIdentityID(
	ctx context.Context,
	identityID identitydomain.ID,
) ([]*sessiondomain.Session, error) {
	rows, err := r.client.Executor(ctx).Query(ctx, `
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
		WHERE s.identity_id = $1
		GROUP BY s.id
		ORDER BY s.created_at DESC, s.id`,
		identityID.UUID(),
	)
	if err != nil {
		return nil, fmt.Errorf("list identity sessions: %w", err)
	}
	defer rows.Close()

	models, err := scanRows(rows)
	if err != nil {
		return nil, err
	}

	entities := make([]*sessiondomain.Session, 0, len(models))
	for _, model := range models {
		entities = append(entities, toDomain(model))
	}

	return entities, nil
}
