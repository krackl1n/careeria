package kafka

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/IBM/sarama"
)

type Producer struct {
	producer sarama.AsyncProducer
	publish  Publisher

	closing chan struct{}
	done    chan struct{}

	once sync.Once
}

type deliveryResult struct {
	done chan error
}

func NewProducer(
	cfg ProducerConfig,
	options ...ProducerOption,
) (*Producer, error) {
	saramaConfig, err := cfg.newSaramaConfig()
	if err != nil {
		return nil, fmt.Errorf("kafka: build producer config: %w", err)
	}

	rawProducer, err := sarama.NewAsyncProducer(
		cfg.Connection.Brokers,
		saramaConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("kafka: create async producer: %w", err)
	}

	resolvedOptions := producerOptions{}

	for _, option := range options {
		if option != nil {
			option(&resolvedOptions)
		}
	}

	producer := &Producer{
		producer: rawProducer,
		closing:  make(chan struct{}),
		done:     make(chan struct{}),
	}

	producer.publish = applyProducerMiddleware(
		PublisherFunc(producer.publishRaw),
		resolvedOptions.middlewares,
	)

	go producer.drainResults()

	return producer, nil
}

func (p *Producer) Publish(
	ctx context.Context,
	message Message,
) error {
	if p == nil {
		return ErrProducerClosed
	}

	if ctx == nil {
		return ErrNilContext
	}

	return p.publish.Publish(ctx, CloneMessage(message))
}

func (p *Producer) Close() error {
	if p == nil {
		return nil
	}

	p.once.Do(func() {
		close(p.closing)
		p.producer.AsyncClose()
		<-p.done
	})

	return nil
}

func (p *Producer) publishRaw(
	ctx context.Context,
	message Message,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if strings.TrimSpace(message.Topic) == "" {
		return errors.New("kafka: message topic is required")
	}

	select {
	case <-p.closing:
		return ErrProducerClosed
	default:
	}

	result := &deliveryResult{
		done: make(chan error, 1),
	}

	record := toSaramaProducerMessage(message, result)

	select {
	case <-p.closing:
		return ErrProducerClosed

	case p.producer.Input() <- record:

	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case err := <-result.done:
		return err

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Producer) drainResults() {
	defer close(p.done)

	successes := p.producer.Successes()
	errorsChannel := p.producer.Errors()

	for successes != nil || errorsChannel != nil {
		select {
		case message, ok := <-successes:
			if !ok {
				successes = nil
				continue
			}

			if message != nil {
				completeDelivery(message.Metadata, nil)
			}

		case producerError, ok := <-errorsChannel:
			if !ok {
				errorsChannel = nil
				continue
			}

			if producerError == nil || producerError.Msg == nil {
				continue
			}

			err := producerError.Err
			if err == nil {
				err = errors.New("unknown producer error")
			}

			completeDelivery(
				producerError.Msg.Metadata,
				fmt.Errorf("kafka: publish message: %w", err),
			)
		}
	}
}

func completeDelivery(
	metadata any,
	err error,
) {
	result, ok := metadata.(*deliveryResult)
	if !ok || result == nil {
		return
	}

	select {
	case result.done <- err:
	default:
	}
}

func toSaramaProducerMessage(
	message Message,
	result *deliveryResult,
) *sarama.ProducerMessage {
	record := &sarama.ProducerMessage{
		Topic:    message.Topic,
		Metadata: result,
	}

	if message.Key != nil {
		record.Key = sarama.ByteEncoder(message.Key)
	}

	if message.Value != nil {
		record.Value = sarama.ByteEncoder(message.Value)
	}

	if len(message.Headers) == 0 {
		return record
	}

	record.Headers = make(
		[]sarama.RecordHeader,
		0,
		len(message.Headers),
	)

	for _, header := range message.Headers {
		record.Headers = append(
			record.Headers,
			sarama.RecordHeader{
				Key:   []byte(header.Key),
				Value: header.Value,
			},
		)
	}

	return record
}
