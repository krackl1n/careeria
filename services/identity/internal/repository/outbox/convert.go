package outbox

import (
	"encoding/json"
	"fmt"
	"slices"

	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
)

func toRow(message eventdomain.Message) (row, error) {
	headers, err := json.Marshal(message.Headers)
	if err != nil {
		return row{}, fmt.Errorf("marshal outbox headers: %w", err)
	}
	return row{
		ID:               message.ID,
		EventID:          message.EventID,
		AggregateType:    message.AggregateType,
		AggregateID:      message.AggregateID,
		AggregateVersion: message.AggregateVersion,
		EventType:        message.EventType,
		Payload:          slices.Clone(message.Payload),
		Headers:          headers,
		PartitionKey:     message.PartitionKey,
		OccurredAt:       message.OccurredAt,
	}, nil
}

func toDomain(model row) (eventdomain.Message, error) {
	var headers map[string]any
	if err := json.Unmarshal(model.Headers, &headers); err != nil {
		return eventdomain.Message{}, fmt.Errorf("unmarshal outbox headers: %w", err)
	}
	return eventdomain.Message{
		ID:               model.ID,
		EventID:          model.EventID,
		AggregateType:    model.AggregateType,
		AggregateID:      model.AggregateID,
		AggregateVersion: model.AggregateVersion,
		EventType:        model.EventType,
		Payload:          slices.Clone(model.Payload),
		Headers:          headers,
		PartitionKey:     model.PartitionKey,
		OccurredAt:       model.OccurredAt,
	}, nil
}
