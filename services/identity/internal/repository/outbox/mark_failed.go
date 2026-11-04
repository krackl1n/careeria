package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	sharedoutbox "github.com/krackl1n/careeria/outbox"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
)

func (r *PostgresRepository) MarkFailed(ctx context.Context, entry sharedoutbox.Entry[eventdomain.Message], cause error) error {
	id, err := uuid.Parse(entry.ID)
	if err != nil {
		return fmt.Errorf("parse outbox message ID: %w", err)
	}
	_, err = r.client.Executor(ctx).Exec(ctx, `
		UPDATE public.outbox_messages
		SET
			status = 'FAILED',
			processing_started_at = NULL,
			next_attempt_at = now() + ($2 * interval '1 second'),
			last_error = $3,
			updated_at = now()
		WHERE id = $1
			AND status = 'PROCESSING'`, id, int64(r.retryDelay/time.Second), cause.Error())
	if err != nil {
		return fmt.Errorf("mark outbox message failed: %w", err)
	}
	return nil
}
