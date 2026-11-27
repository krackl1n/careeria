package outbox

import (
	"context"
	"fmt"

	"github.com/krackl1n/careeria/services/vacancy/internal/domain/event"
)

func (r *PostgresRepository) Create(ctx context.Context, message event.Message) error {
	_, err := r.client.Executor(ctx).Exec(ctx, `
		INSERT INTO public.outbox_messages
		(id,event_id,aggregate_type,aggregate_id,aggregate_version,event_type,
		 payload,headers,partition_key,status,occurred_at,created_at,updated_at)
		VALUES ($1,$2,'vacancy',$3,$4,$5,$6,'{}'::jsonb,$7,'PENDING',$8,$8,$8)`,
		message.ID, message.EventID, message.AggregateID, message.AggregateVersion,
		message.EventType, message.Payload, message.AggregateID.String(), message.OccurredAt)
	if err != nil {
		return fmt.Errorf("create vacancy outbox message: %w", err)
	}
	return nil
}
