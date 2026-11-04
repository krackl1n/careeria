package kafka

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const HeaderEventType = "x-event-type"

type EventEnvelope struct {
	MessageID        uuid.UUID       `json:"message_id"`
	EventID          uuid.UUID       `json:"event_id"`
	EventType        string          `json:"event_type"`
	AggregateType    string          `json:"aggregate_type"`
	AggregateID      uuid.UUID       `json:"aggregate_id"`
	AggregateVersion *int64          `json:"aggregate_version,omitempty"`
	OccurredAt       time.Time       `json:"occurred_at"`
	Payload          json.RawMessage `json:"payload"`
	Headers          map[string]any  `json:"headers,omitempty"`
}
