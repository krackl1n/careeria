package kafka

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEventEnvelopeJSONContract(t *testing.T) {
	messageID := uuid.MustParse("00000000-0000-7000-8000-000000000001")
	eventID := uuid.MustParse("00000000-0000-7000-8000-000000000002")
	aggregateID := uuid.MustParse("00000000-0000-7000-8000-000000000003")
	version := int64(4)
	envelope := EventEnvelope{
		MessageID: messageID, EventID: eventID, EventType: "meeting.scheduled",
		AggregateType: "meeting", AggregateID: aggregateID, AggregateVersion: &version,
		OccurredAt: time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC),
		Payload:    json.RawMessage(`{"tenant_id":"tenant"}`), Headers: map[string]any{"source": "meeting"},
	}

	body, err := json.Marshal(envelope)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(body, &decoded))
	require.Equal(t, messageID.String(), decoded["message_id"])
	require.Equal(t, eventID.String(), decoded["event_id"])
	require.Equal(t, "meeting.scheduled", decoded["event_type"])
	require.Equal(t, aggregateID.String(), decoded["aggregate_id"])
	require.Equal(t, float64(version), decoded["aggregate_version"])
	require.Equal(t, map[string]any{"tenant_id": "tenant"}, decoded["payload"])
}
