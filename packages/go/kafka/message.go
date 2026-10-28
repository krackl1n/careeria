package kafka

import (
	"context"
	"slices"
)

const (
	HeaderMessageID     = "x-message-id"
	HeaderCorrelationID = "x-correlation-id"
	HeaderCausationID   = "x-causation-id"
)

type Header struct {
	Key   string
	Value []byte
}

type Message struct {
	Topic   string
	Key     []byte
	Value   []byte
	Headers []Header
}

type ConsumedMessage struct {
	Topic     string
	Partition int32
	Offset    int64

	Key     []byte
	Value   []byte
	Headers []Header
}

type Publisher interface {
	Publish(ctx context.Context, message Message) error
}

// Handler may be called concurrently for different Kafka partitions.
type Handler interface {
	Handle(ctx context.Context, message ConsumedMessage) error
}

type HandlerFunc func(ctx context.Context, message ConsumedMessage) error

func (f HandlerFunc) Handle(
	ctx context.Context,
	message ConsumedMessage,
) error {
	return f(ctx, message)
}

func HeaderValue(headers []Header, key string) ([]byte, bool) {
	for _, header := range headers {
		if header.Key == key {
			return header.Value, true
		}
	}

	return nil, false
}

func HeaderString(headers []Header, key string) (string, bool) {
	value, ok := HeaderValue(headers, key)
	if !ok {
		return "", false
	}

	return string(value), true
}

func SetHeader(headers []Header, key string, value []byte) []Header {
	valueCopy := append([]byte(nil), value...)

	for index := range headers {
		if headers[index].Key == key {
			headers[index].Value = valueCopy
			return headers
		}
	}

	return append(headers, Header{
		Key:   key,
		Value: valueCopy,
	})
}

// CloneMessage returns a deep copy of a producer message.
func CloneMessage(message Message) Message {
	result := Message{
		Topic: message.Topic,
		Key:   slices.Clone(message.Key),
		Value: slices.Clone(message.Value),
	}

	if len(message.Headers) == 0 {
		return result
	}

	result.Headers = make([]Header, 0, len(message.Headers))
	for _, header := range message.Headers {
		result.Headers = append(result.Headers, Header{
			Key:   header.Key,
			Value: slices.Clone(header.Value),
		})
	}

	return result
}
