package outbox

import (
	"context"
	"fmt"

	sharedoutbox "github.com/krackl1n/careeria/outbox"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
)

func (r *PostgresRepository) ClaimPending(ctx context.Context, limit int) ([]sharedoutbox.Entry[eventdomain.Message], error) {
	rows, err := r.client.Executor(ctx).Query(ctx, `
		WITH claimed AS (
			SELECT id
			FROM public.outbox_messages
			WHERE status IN ('PENDING', 'FAILED')
				AND (next_attempt_at IS NULL OR next_attempt_at <= now())
			ORDER BY occurred_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		UPDATE public.outbox_messages AS o
		SET
			status = 'PROCESSING',
			attempts = attempts + 1,
			processing_started_at = now(),
			updated_at = now()
		FROM claimed
		WHERE o.id = claimed.id
		RETURNING
			o.id, o.event_id, o.aggregate_type, o.aggregate_id,
			o.aggregate_version, o.event_type, o.payload, o.headers,
			o.partition_key, o.occurred_at`, limit)
	if err != nil {
		return nil, fmt.Errorf("claim outbox messages: %w", err)
	}
	defer rows.Close()

	models, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	entries := make([]sharedoutbox.Entry[eventdomain.Message], 0, len(models))
	for _, model := range models {
		message, err := toDomain(model)
		if err != nil {
			return nil, err
		}
		entries = append(entries, sharedoutbox.Entry[eventdomain.Message]{
			ID: message.ID.String(), ClaimToken: message.ID.String(), Message: message,
		})
	}
	return entries, nil
}
