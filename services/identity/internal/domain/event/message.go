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
	return NewMessage(
		f.ids.New(),
		f.ids.New(),
		eventType,
		aggregateType,
		aggregateID,
		aggregateVersion,
		payload,
		occurredAt,
	)
}

func NewMessage(
	messageID, eventID uuid.UUID,
	eventType Type,
	aggregateType AggregateType,
	aggregateID uuid.UUID,
	aggregateVersion *int64,
	payload any,
	occurredAt time.Time,
) (Message, error) {
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return Message{}, fmt.Errorf("marshal %s event payload: %w", eventType, err)
	}

	partitionKey := aggregateID.String()
	return Message{
		ID:               messageID,
		EventID:          eventID,
		AggregateType:    aggregateType,
		AggregateID:      aggregateID,
		AggregateVersion: aggregateVersion,
		EventType:        eventType,
		Payload:          encodedPayload,
		Headers:          map[string]any{},
		PartitionKey:     &partitionKey,
		OccurredAt:       occurredAt,
	}, nil
}
