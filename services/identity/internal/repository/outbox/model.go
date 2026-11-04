package outbox

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
)

type row struct {
	ID               uuid.UUID
	EventID          uuid.UUID
	AggregateType    eventdomain.AggregateType
	AggregateID      uuid.UUID
	AggregateVersion *int64
	EventType        eventdomain.Type
	Payload          []byte
	Headers          []byte
	PartitionKey     *string
	OccurredAt       time.Time
}

func scanRow(source pgx.Row) (row, error) {
	var model row
	if err := source.Scan(
		&model.ID,
		&model.EventID,
		&model.AggregateType,
		&model.AggregateID,
		&model.AggregateVersion,
		&model.EventType,
		&model.Payload,
		&model.Headers,
		&model.PartitionKey,
		&model.OccurredAt,
	); err != nil {
		return row{}, fmt.Errorf("scan outbox message row: %w", err)
	}
	return model, nil
}

func scanRows(rows pgx.Rows) ([]row, error) {
	models := make([]row, 0)
	for rows.Next() {
		model, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		models = append(models, model)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox messages: %w", err)
	}
	return models, nil
}
