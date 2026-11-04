package app

import (
	"context"
	"fmt"

	sharedkafka "github.com/krackl1n/careeria/kafka"
	producermw "github.com/krackl1n/careeria/kafka/middleware/producer"
	kafkapreset "github.com/krackl1n/careeria/kafka/preset"
	sharedoutbox "github.com/krackl1n/careeria/outbox"
	sharedruntime "github.com/krackl1n/careeria/runtime"
	eventdomain "github.com/krackl1n/careeria/services/identity/internal/domain/event"
	kafkaintegration "github.com/krackl1n/careeria/services/identity/internal/integrations/kafka"
)

const identityProducerName = "identity-outbox"

func (a *App) KafkaProducer() (*sharedkafka.Producer, error) {
	if a.kafkaProducer != nil {
		return a.kafkaProducer, nil
	}
	metrics, err := a.Metrics()
	if err != nil {
		return nil, err
	}
	producerMetrics, err := producermw.NewMetrics(producermw.MetricsConfig{
		Registerer: metrics.Registry(), Namespace: metrics.Namespace(), Subsystem: "identity_kafka_producer", Producer: identityProducerName,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize producer metrics: %w", err)
	}
	config := a.dependencies.Config.KafkaProducer
	middleware := kafkapreset.Producer(kafkapreset.ProducerConfig{
		Logger: a.dependencies.Logger, Producer: identityProducerName, Metrics: producerMetrics,
		Timeout:    &producermw.TimeoutConfig{Provider: producermw.StaticTimeout(a.dependencies.Config.Outbox.PublishTimeout)},
		Validation: producermw.ValidationConfig{Validator: producermw.DefaultEnvelopeValidator{RequireValue: true, MaxValueBytes: config.MaxMessageBytes}},
	})
	producer, err := sharedkafka.NewProducer(config, sharedkafka.WithProducerMiddleware(middleware...))
	if err != nil {
		return nil, err
	}
	a.dependencies.Closer.Add(producer)
	a.kafkaProducer = producer
	return producer, nil
}

func (a *App) EventPublisher() (*kafkaintegration.Publisher, error) {
	if a.eventPublisher == nil {
		producer, err := a.KafkaProducer()
		if err != nil {
			return nil, err
		}
		a.eventPublisher = kafkaintegration.NewPublisher(producer, a.dependencies.Config.IdentityTopic)
	}
	return a.eventPublisher, nil
}

func (a *App) OutboxWorker() (*sharedoutbox.Worker[eventdomain.Message], error) {
	if a.outboxWorker == nil {
		publisher, err := a.EventPublisher()
		if err != nil {
			return nil, err
		}
		worker, err := sharedoutbox.New(a.dependencies.Config.Outbox, a.Outbox(), publisher, a.dependencies.Logger)
		if err != nil {
			return nil, err
		}
		a.outboxWorker = worker
	}
	return a.outboxWorker, nil
}

func (a *App) OutboxComponent(ctx context.Context) (sharedruntime.Component, error) {
	worker, err := a.OutboxWorker()
	if err != nil {
		return nil, err
	}
	return sharedruntime.NewContextComponent(ctx, worker)
}
