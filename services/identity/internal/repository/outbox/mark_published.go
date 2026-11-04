package outbox

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	sharedoutbox "github.com/krackl1n/careeria/outbox"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
)

func (r *PostgresRepository) MarkPublished(ctx context.Context, entries []sharedoutbox.Entry[eventdomain.Message]) error {
	if len(entries) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(entries))
	for index, entry := range entries {
		id, err := uuid.Parse(entry.ID)
		if err != nil {
			return fmt.Errorf("parse outbox message ID: %w", err)
		}
		ids[index] = id
	}
	_, err := r.client.Executor(ctx).Exec(ctx, `
		UPDATE public.outbox_messages
		SET
			status = 'PUBLISHED',
			published_at = now(),
			processing_started_at = NULL,
			next_attempt_at = NULL,
			last_error = NULL,
			updated_at = now()
		WHERE id = ANY($1)
			AND status = 'PROCESSING'`, ids)
	if err != nil {
		return fmt.Errorf("mark outbox messages published: %w", err)
	}
	return nil
}
