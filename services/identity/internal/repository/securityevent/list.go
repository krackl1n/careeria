package securityevent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/securityevent"
)

func (r *PostgresRepository) List(
	ctx context.Context,
	filter eventdomain.Filter,
) ([]*eventdomain.SecurityEvent, error) {
	var identityID *uuid.UUID

	if filter.IdentityID != nil {
		v := filter.IdentityID.UUID()
		identityID = &v
	}

	limit := filter.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	rows, err := r.client.Executor(ctx).Query(ctx, `
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
		WHERE ($1::uuid IS NULL OR identity_id = $1)
			AND ($2::uuid IS NULL OR correlation_id = $2)
		ORDER BY occurred_at DESC, id
		LIMIT $3 OFFSET $4`,
		identityID,
		filter.CorrelationID,
		limit,
		filter.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list security events: %w", err)
	}
	defer rows.Close()

	models, err := scanRows(rows)
	if err != nil {
		return nil, err
	}

	events := make([]*eventdomain.SecurityEvent, 0, len(models))
	for _, m := range models {
		events = append(events, toDomain(m))
	}

	return events, nil
}
