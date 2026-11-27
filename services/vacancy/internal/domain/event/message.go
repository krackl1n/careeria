package event

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID               uuid.UUID
	EventID          uuid.UUID
	AggregateID      uuid.UUID
	AggregateVersion int64
	EventType        string
	Payload          json.RawMessage
	OccurredAt       time.Time
}
