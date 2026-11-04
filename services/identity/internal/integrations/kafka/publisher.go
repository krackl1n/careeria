package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	sharedkafka "github.com/krackl1n/careeria/kafka"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
)

type Publisher struct {
	publisher sharedkafka.Publisher
	topic     string
}

func NewPublisher(publisher sharedkafka.Publisher, topic string) *Publisher {
	return &Publisher{publisher: publisher, topic: topic}
}

func (p *Publisher) Publish(ctx context.Context, message eventdomain.Message) error {
	body, err := json.Marshal(sharedkafka.EventEnvelope{
		MessageID: message.ID, EventID: message.EventID, EventType: string(message.EventType),
		AggregateType: string(message.AggregateType), AggregateID: message.AggregateID,
		AggregateVersion: message.AggregateVersion, OccurredAt: message.OccurredAt,
		Payload: message.Payload, Headers: message.Headers,
	})
	if err != nil {
		return fmt.Errorf("marshal identity event: %w", err)
	}
	key := message.AggregateID.String()
	if message.PartitionKey != nil && strings.TrimSpace(*message.PartitionKey) != "" {
		key = *message.PartitionKey
	}
	return p.publisher.Publish(ctx, sharedkafka.Message{
		Topic: p.topic, Key: []byte(key), Value: body,
		Headers: []sharedkafka.Header{
			{Key: sharedkafka.HeaderMessageID, Value: []byte(message.ID.String())},
			{Key: sharedkafka.HeaderEventType, Value: []byte(message.EventType)},
		},
	})
}
