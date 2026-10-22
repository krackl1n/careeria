package event

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Type string
type AggregateType string

type IDGenerator interface {
	New() uuid.UUID
}

type Factory struct {
	ids IDGenerator
}

func NewFactory(ids IDGenerator) Factory {
	return Factory{ids: ids}
}

type Message struct {
	ID      uuid.UUID
	EventID uuid.UUID

	AggregateType    AggregateType
	AggregateID      uuid.UUID
	AggregateVersion *int64

	EventType Type
	Payload   json.RawMessage

	Headers      map[string]any
	PartitionKey *string

	OccurredAt time.Time
}

func (f Factory) Create(
	eventType Type,
	aggregateType AggregateType,
	aggregateID uuid.UUID,
	aggregateVersion *int64,
	payload any,
	occurredAt time.Time,
) (Message, error) {
	if eventType == "" {
		return Message{}, fmt.Errorf("event type is required")
	}
	if aggregateType == "" {
		return Message{}, fmt.Errorf("aggregate type is required")
	}
	if aggregateID == uuid.Nil {
		return Message{}, fmt.Errorf("aggregate ID is required")
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Message{}, fmt.Errorf("marshal event payload: %w", err)
	}

	eventID := uuid.New()
	if f.ids != nil {
		eventID = f.ids.New()
	}

	partitionKey := aggregateID.String()

	return Message{
		ID:               eventID,
		EventID:          eventID,
		AggregateType:    aggregateType,
		AggregateID:      aggregateID,
		AggregateVersion: aggregateVersion,
		EventType:        eventType,
		Payload:          payloadBytes,
		Headers: map[string]any{
			"aggregate_type": string(aggregateType),
			"event_type":     string(eventType),
		},
		PartitionKey: &partitionKey,
		OccurredAt:   occurredAt,
	}, nil
}
