package kafka

import "errors"

var (
	ErrNilContext             = errors.New("kafka: context is nil")
	ErrProducerClosed         = errors.New("kafka: producer is closed")
	ErrConsumerClosed         = errors.New("kafka: consumer is closed")
	ErrConsumerAlreadyStarted = errors.New("kafka: consumer is already started")
	ErrHandlerRequired        = errors.New("kafka: handler is required")
)
