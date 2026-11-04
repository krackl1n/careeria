package kafka

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/IBM/sarama"
)

type Consumer struct {
	name   string
	group  sarama.ConsumerGroup
	topics []string

	handler *consumerGroupHandler

	started atomic.Bool
	closed  atomic.Bool

	shutdownOnce sync.Once
	shutdownDone chan struct{}
	shutdownErr  error
}

func NewConsumer(
	cfg ConsumerConfig,
	handler Handler,
	options ...ConsumerOption,
) (*Consumer, error) {
	if handler == nil {
		return nil, ErrHandlerRequired
	}

	if len(cfg.Topics) == 0 {
		return nil, errors.New("kafka: consumer topics are required")
	}

	saramaConfig, err := cfg.newSaramaConfig()
	if err != nil {
		return nil, fmt.Errorf("kafka: build consumer config: %w", err)
	}

	group, err := sarama.NewConsumerGroup(
		cfg.Connection.Brokers,
		cfg.GroupID,
		saramaConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("kafka: create consumer group: %w", err)
	}

	resolvedOptions := consumerOptions{}

	for _, option := range options {
		if option != nil {
			option(&resolvedOptions)
		}
	}

	wrappedHandler := applyConsumerMiddleware(
		handler,
		resolvedOptions.middlewares,
	)

	consumer := &Consumer{
		name:  cfg.GroupID,
		group: group,
		topics: append(
			[]string(nil),
			cfg.Topics...,
		),
		handler: &consumerGroupHandler{
			handler: wrappedHandler,
		},
		shutdownDone: make(chan struct{}),
	}

	return consumer, nil
}

func (c *Consumer) Name() string {
	if c == nil {
		return ""
	}
	return c.name
}

func (c *Consumer) Start(ctx context.Context) error {
	if c == nil {
		return ErrConsumerClosed
	}

	if ctx == nil {
		return ErrNilContext
	}

	if c.closed.Load() {
		return ErrConsumerClosed
	}

	if !c.started.CompareAndSwap(false, true) {
		return ErrConsumerAlreadyStarted
	}

	defer c.started.Store(false)

	for {
		c.handler.reset()

		err := c.group.Consume(
			ctx,
			c.topics,
			c.handler,
		)

		if handlerErr := c.handler.err(); handlerErr != nil {
			return handlerErr
		}

		if c.closed.Load() {
			return nil
		}

		if ctx.Err() != nil {
			return nil
		}

		if err != nil {
			if errors.Is(err, sarama.ErrClosedConsumerGroup) {
				return nil
			}

			return fmt.Errorf("kafka: consume group: %w", err)
		}
	}
}

func (c *Consumer) Shutdown(ctx context.Context) error {
	if c == nil {
		return nil
	}

	if ctx == nil {
		return ErrNilContext
	}

	c.shutdownOnce.Do(func() {
		c.closed.Store(true)

		go func() {
			c.shutdownErr = c.group.Close()
			close(c.shutdownDone)
		}()
	})

	select {
	case <-c.shutdownDone:
		return c.shutdownErr

	case <-ctx.Done():
		return ctx.Err()
	}
}

type consumerGroupHandler struct {
	handler Handler

	mu       sync.Mutex
	firstErr error
}

func (h *consumerGroupHandler) Setup(
	session sarama.ConsumerGroupSession,
) error {
	return nil
}

func (h *consumerGroupHandler) Cleanup(
	session sarama.ConsumerGroupSession,
) error {
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	for {
		select {
		case <-session.Context().Done():
			return nil

		case record, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			message := fromSaramaConsumerMessage(record)

			err := h.handler.Handle(
				session.Context(),
				message,
			)
			if err != nil {
				if session.Context().Err() != nil {
					return nil
				}

				wrappedErr := fmt.Errorf(
					"kafka: handle message topic=%s partition=%d offset=%d: %w",
					record.Topic,
					record.Partition,
					record.Offset,
					err,
				)

				h.setErr(wrappedErr)

				return wrappedErr
			}

			if session.Context().Err() != nil {
				return nil
			}

			session.MarkMessage(record, "")
		}
	}
}

func (h *consumerGroupHandler) reset() {
	h.mu.Lock()
	h.firstErr = nil
	h.mu.Unlock()
}

func (h *consumerGroupHandler) setErr(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.firstErr == nil {
		h.firstErr = err
	}
}

func (h *consumerGroupHandler) err() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.firstErr
}

func fromSaramaConsumerMessage(
	record *sarama.ConsumerMessage,
) ConsumedMessage {
	if record == nil {
		return ConsumedMessage{}
	}

	message := ConsumedMessage{
		Topic:     record.Topic,
		Partition: record.Partition,
		Offset:    record.Offset,
		Key:       slices.Clone(record.Key),
		Value:     slices.Clone(record.Value),
	}

	if len(record.Headers) == 0 {
		return message
	}

	message.Headers = make(
		[]Header,
		0,
		len(record.Headers),
	)

	for _, header := range record.Headers {
		if header == nil {
			continue
		}

		message.Headers = append(
			message.Headers,
			Header{
				Key:   string(header.Key),
				Value: slices.Clone(header.Value),
			},
		)
	}

	return message
}
