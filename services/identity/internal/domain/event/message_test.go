package event_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/krackl1n/careeria/services/identity/internal/domain/event"
	"github.com/stretchr/testify/require"
)

type ids struct {
	values []uuid.UUID
}

func (g *ids) New() uuid.UUID {
	value := g.values[0]
	g.values = g.values[1:]
	return value
}

func TestFactoryCreatesIdentityEventEnvelope(t *testing.T) {
	t.Parallel()

	messageID, eventID, aggregateID := uuid.New(), uuid.New(), uuid.New()
	occurredAt := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	message, err := event.NewFactory(&ids{values: []uuid.UUID{messageID, eventID}}).Create(
		event.Type("IdentityCreated"),
		event.AggregateType("identity"),
		aggregateID,
		nil,
		struct {
			IdentityID uuid.UUID `json:"identity_id"`
		}{IdentityID: aggregateID},
		occurredAt,
	)
	require.NoError(t, err)
	require.Equal(t, messageID, message.ID)
	require.Equal(t, eventID, message.EventID)
	require.Equal(t, aggregateID, message.AggregateID)
	require.Equal(t, aggregateID.String(), *message.PartitionKey)
	require.JSONEq(t, `{"identity_id":"`+aggregateID.String()+`"}`, string(message.Payload))
}
