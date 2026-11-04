package securityevent

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/securityevent"
)

func (r *PostgresRepository) Get(
	ctx context.Context,
	id eventdomain.ID,
) (*eventdomain.SecurityEvent, error) {
	m, err := scanRow(
		r.client.Executor(ctx).QueryRow(ctx, `
			SELECT
				id,
				event_type,
				result,
				actor_type,
				identity_id,
				service_account_id,
				session_id,
				oauth_client_id,
				identifier_hash,
				ip_address,
				user_agent,
				correlation_id,
				occurred_at,
				metadata
			FROM public.security_events
			WHERE id = $1`,
			id.UUID(),
		),
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, eventdomain.ErrNotFound
		}

		return nil, fmt.Errorf("get security event: %w", err)
	}

	return toDomain(m), nil
}
