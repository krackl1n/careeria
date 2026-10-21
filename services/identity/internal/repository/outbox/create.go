package outbox

import (
	"context"
	"fmt"

	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
)

func (r *PostgresRepository) Create(ctx context.Context, message eventdomain.Message) error {
	model, err := toRow(message)
	if err != nil {
		return err
	}
	_, err = r.client.Executor(ctx).Exec(ctx, `
		INSERT INTO public.outbox_messages (
			id, event_id, aggregate_type, aggregate_id, aggregate_version,
			event_type, payload, headers, partition_key, status, occurred_at,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'PENDING', $10, $10, $10)`,
		model.ID, model.EventID, model.AggregateType, model.AggregateID, model.AggregateVersion,
		model.EventType, model.Payload, model.Headers, model.PartitionKey, model.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf("create outbox message: %w", err)
	}
	return nil
}
