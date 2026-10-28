package kafka

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"
)

type mockAsyncProducer struct {
	input     chan *sarama.ProducerMessage
	successes chan *sarama.ProducerMessage
	errors    chan *sarama.ProducerError

	asyncClosed bool
}

func (m *mockAsyncProducer) AsyncClose() {
	if m.asyncClosed {
		return
	}
	m.asyncClosed = true
	close(m.successes)
	close(m.errors)
}

func (m *mockAsyncProducer) Close() error                              { return nil }
func (m *mockAsyncProducer) Input() chan<- *sarama.ProducerMessage     { return m.input }
func (m *mockAsyncProducer) Successes() <-chan *sarama.ProducerMessage { return m.successes }
func (m *mockAsyncProducer) Errors() <-chan *sarama.ProducerError      { return m.errors }
func (m *mockAsyncProducer) IsTransactional() bool                     { return false }
func (m *mockAsyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag   { return 0 }
func (m *mockAsyncProducer) BeginTxn() error                           { return nil }
func (m *mockAsyncProducer) CommitTxn() error                          { return nil }
func (m *mockAsyncProducer) AbortTxn() error                           { return nil }
func (m *mockAsyncProducer) AddOffsetsToTxn(map[string][]*sarama.PartitionOffsetMetadata, string) error {
	return nil
}
func (m *mockAsyncProducer) AddOffsetsToTxnWithGroupMetadata(map[string][]*sarama.PartitionOffsetMetadata, *sarama.ConsumerGroupMetadata) error {
	return nil
}
func (m *mockAsyncProducer) AddMessageToTxn(*sarama.ConsumerMessage, string, *string) error {
	return nil
}
func (m *mockAsyncProducer) AddMessageToTxnWithGroupMetadata(*sarama.ConsumerMessage, *sarama.ConsumerGroupMetadata, *string) error {
	return nil
}

type mockConsumerGroup struct {
	consumeFn func(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error
	closeFn   func() error
}

func (m *mockConsumerGroup) Consume(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
	return m.consumeFn(ctx, topics, handler)
}
func (m *mockConsumerGroup) Errors() <-chan error { return nil }
func (m *mockConsumerGroup) Close() error {
	if m.closeFn != nil {
		return m.closeFn()
	}
	return nil
}
func (m *mockConsumerGroup) Pause(map[string][]int32)  {}
func (m *mockConsumerGroup) Resume(map[string][]int32) {}
func (m *mockConsumerGroup) PauseAll()                 {}
func (m *mockConsumerGroup) ResumeAll()                {}

type mockSession struct {
	ctx    context.Context
	marked []*sarama.ConsumerMessage
}

func (m *mockSession) Claims() map[string][]int32               { return nil }
func (m *mockSession) MemberID() string                         { return "" }
func (m *mockSession) GenerationID() int32                      { return 0 }
func (m *mockSession) MarkOffset(string, int32, int64, string)  {}
func (m *mockSession) Commit()                                  {}
func (m *mockSession) ResetOffset(string, int32, int64, string) {}
func (m *mockSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {
	m.marked = append(m.marked, msg)
}
func (m *mockSession) Context() context.Context { return m.ctx }

type mockClaim struct {
	messages chan *sarama.ConsumerMessage
}

func (m *mockClaim) Topic() string                            { return "topic" }
func (m *mockClaim) Partition() int32                         { return 1 }
func (m *mockClaim) InitialOffset() int64                     { return 0 }
func (m *mockClaim) HighWaterMarkOffset() int64               { return 0 }
func (m *mockClaim) Messages() <-chan *sarama.ConsumerMessage { return m.messages }

func TestConnectionConfigValidateAndSaramaConfig(t *testing.T) {
	t.Parallel()

	cfg := ConnectionConfig{
		Brokers:  []string{"localhost:9092"},
		ClientID: "svc",
		Version:  "2.8.0",
		SASL: SASLConfig{
			Enabled:   true,
			Mechanism: SASLMechanismPlain,
			Username:  "user",
			Password:  "pass",
		},
	}

	require.NoError(t, cfg.validate())

	saramaCfg, err := cfg.newSaramaConfig()
	require.NoError(t, err)
	require.Equal(t, "svc", saramaCfg.ClientID)
	require.True(t, saramaCfg.Net.SASL.Enable)
	require.Equal(t, string(sarama.SASLTypePlaintext), string(saramaCfg.Net.SASL.Mechanism))
}

func TestProducerAndConsumerSaramaConfig(t *testing.T) {
	t.Parallel()

	connection := ConnectionConfig{
		Brokers:  []string{"localhost:9092"},
		ClientID: "svc",
		Version:  "2.8.0",
	}

	producerCfg, err := (ProducerConfig{
		Connection:  connection,
		Compression: CompressionGZIP,
	}).newSaramaConfig()
	require.NoError(t, err)
	require.True(t, producerCfg.Producer.Return.Successes)
	require.Equal(t, sarama.CompressionGZIP, producerCfg.Producer.Compression)
	require.True(t, producerCfg.Producer.Idempotent)

	consumerCfg, err := (ConsumerConfig{
		Connection: connection,
		GroupID:    "group",
		Topics:     []string{"orders"},
	}).newSaramaConfig()
	require.NoError(t, err)
	require.Equal(t, sarama.OffsetOldest, consumerCfg.Consumer.Offsets.Initial)
	require.Equal(t, defaultConsumerCommitInterval, consumerCfg.Consumer.Offsets.AutoCommit.Interval)
}

func TestConversionHelpers(t *testing.T) {
	t.Parallel()

	compression, err := toSaramaCompression(CompressionZSTD)
	require.NoError(t, err)
	require.Equal(t, sarama.CompressionZSTD, compression)

	_, err = toSaramaCompression("bad")
	require.EqualError(t, err, `kafka: unsupported compression "bad"`)

	offset, err := toSaramaInitialOffset(InitialOffsetNewest)
	require.NoError(t, err)
	require.Equal(t, sarama.OffsetNewest, offset)

	_, err = toSaramaInitialOffset("bad")
	require.EqualError(t, err, `kafka: unsupported initial offset "bad"`)

	require.Equal(t, 10, defaultInt(0, 10))
	require.Equal(t, 20, defaultInt(20, 10))
	require.Equal(t, time.Second, defaultDuration(0, time.Second))
}

func TestSecurityHelpers(t *testing.T) {
	t.Parallel()

	require.NoError(t, TLSConfig{}.validate())
	require.EqualError(t, TLSConfig{Enabled: true}.validate(), "kafka: tls server name is required when certificate verification is enabled")
	require.EqualError(t, SASLConfig{Enabled: true}.validate(), "kafka: sasl username is required")
	require.NoError(t, SASLConfig{
		Enabled:   true,
		Mechanism: SASLMechanismSCRAMSHA256,
		Username:  "user",
		Password:  "pass",
	}.validate())

	tlsCfg, err := buildTLSConfig(TLSConfig{
		Enabled:            true,
		ServerName:         "localhost",
		InsecureSkipVerify: true,
	})
	require.NoError(t, err)
	require.NotNil(t, tlsCfg)
	require.Equal(t, uint16(tls.VersionTLS12), tlsCfg.MinVersion)

	saramaCfg := sarama.NewConfig()
	require.NoError(t, applySASL(saramaCfg, SASLConfig{
		Enabled:   true,
		Mechanism: SASLMechanismSCRAMSHA512,
		Username:  "user",
		Password:  "pass",
	}))
	require.Equal(t, string(sarama.SASLTypeSCRAMSHA512), string(saramaCfg.Net.SASL.Mechanism))
	require.NotNil(t, saramaCfg.Net.SASL.SCRAMClientGeneratorFunc)
}

func TestMessageHelpers(t *testing.T) {
	t.Parallel()

	headers := []Header{{Key: "x-id", Value: []byte("1")}}
	value, ok := HeaderValue(headers, "x-id")
	require.True(t, ok)
	require.Equal(t, []byte("1"), value)

	str, ok := HeaderString(headers, "x-id")
	require.True(t, ok)
	require.Equal(t, "1", str)

	updated := SetHeader(headers, "x-id", []byte("2"))
	require.Equal(t, []byte("2"), updated[0].Value)

	appended := SetHeader(updated, "x-role", []byte("admin"))
	require.Len(t, appended, 2)

	message := Message{
		Topic:   "orders",
		Key:     []byte("key"),
		Value:   []byte("value"),
		Headers: appended,
	}
	cloned := CloneMessage(message)
	cloned.Key[0] = 'K'
	cloned.Headers[0].Value[0] = '9'
	require.Equal(t, []byte("key"), message.Key)
	require.Equal(t, []byte("2"), message.Headers[0].Value)
}

func TestMiddlewareOrder(t *testing.T) {
	t.Parallel()

	var producerOrder []string
	publisher := applyProducerMiddleware(PublisherFunc(func(ctx context.Context, message Message) error {
		producerOrder = append(producerOrder, "publish")
		return nil
	}), []ProducerMiddleware{
		func(next Publisher) Publisher {
			return PublisherFunc(func(ctx context.Context, message Message) error {
				producerOrder = append(producerOrder, "first")
				return next.Publish(ctx, message)
			})
		},
		func(next Publisher) Publisher {
			return PublisherFunc(func(ctx context.Context, message Message) error {
				producerOrder = append(producerOrder, "second")
				return next.Publish(ctx, message)
			})
		},
	})
	require.NoError(t, publisher.Publish(context.Background(), Message{}))
	require.Equal(t, []string{"first", "second", "publish"}, producerOrder)

	var consumerOrder []string
	handler := applyConsumerMiddleware(HandlerFunc(func(ctx context.Context, message ConsumedMessage) error {
		consumerOrder = append(consumerOrder, "handle")
		return nil
	}), []ConsumerMiddleware{
		func(next Handler) Handler {
			return HandlerFunc(func(ctx context.Context, message ConsumedMessage) error {
				consumerOrder = append(consumerOrder, "first")
				return next.Handle(ctx, message)
			})
		},
	})
	require.NoError(t, handler.Handle(context.Background(), ConsumedMessage{}))
	require.Equal(t, []string{"first", "handle"}, consumerOrder)
}

func TestPublisherFunc(t *testing.T) {
	t.Parallel()

	called := false
	message := Message{Topic: "orders"}

	err := PublisherFunc(func(ctx context.Context, got Message) error {
		called = true
		require.Equal(t, message, got)
		return nil
	}).Publish(context.Background(), message)

	require.NoError(t, err)
	require.True(t, called)
}

func TestProducerPublishAndClose(t *testing.T) {
	t.Parallel()

	raw := &mockAsyncProducer{
		input:     make(chan *sarama.ProducerMessage, 1),
		successes: make(chan *sarama.ProducerMessage, 1),
		errors:    make(chan *sarama.ProducerError, 1),
	}

	producer := &Producer{
		producer: raw,
		closing:  make(chan struct{}),
		done:     make(chan struct{}),
	}
	producer.publish = PublisherFunc(producer.publishRaw)
	go producer.drainResults()

	done := make(chan error, 1)
	go func() {
		done <- producer.Publish(context.Background(), Message{
			Topic:   "orders",
			Key:     []byte("key"),
			Value:   []byte("value"),
			Headers: []Header{{Key: "x-id", Value: []byte("1")}},
		})
	}()

	record := <-raw.input
	require.Equal(t, "orders", record.Topic)
	require.Len(t, record.Headers, 1)
	raw.successes <- record

	require.NoError(t, <-done)
	require.NoError(t, producer.Close())
}

func TestProducerEdgeCases(t *testing.T) {
	t.Parallel()

	require.Equal(t, ErrProducerClosed, (*Producer)(nil).Publish(context.Background(), Message{}))

	raw := &mockAsyncProducer{
		input:     make(chan *sarama.ProducerMessage, 1),
		successes: make(chan *sarama.ProducerMessage),
		errors:    make(chan *sarama.ProducerError),
	}
	producer := &Producer{
		producer: raw,
		closing:  make(chan struct{}),
		done:     make(chan struct{}),
	}
	producer.publish = PublisherFunc(producer.publishRaw)

	var nilCtx context.Context
	require.Equal(t, ErrNilContext, producer.Publish(nilCtx, Message{}))
	require.EqualError(t, producer.Publish(context.Background(), Message{}), "kafka: message topic is required")

	close(producer.closing)
	require.Equal(t, ErrProducerClosed, producer.Publish(context.Background(), Message{Topic: "orders"}))
}

func TestDrainResultsAndCompleteDelivery(t *testing.T) {
	t.Parallel()

	raw := &mockAsyncProducer{
		input:     make(chan *sarama.ProducerMessage),
		successes: make(chan *sarama.ProducerMessage, 1),
		errors:    make(chan *sarama.ProducerError, 1),
	}
	producer := &Producer{
		producer: raw,
		done:     make(chan struct{}),
	}

	successResult := &deliveryResult{done: make(chan error, 1)}
	errorResult := &deliveryResult{done: make(chan error, 1)}

	go producer.drainResults()
	raw.successes <- &sarama.ProducerMessage{Metadata: successResult}
	raw.errors <- &sarama.ProducerError{
		Msg: &sarama.ProducerMessage{Metadata: errorResult},
		Err: errors.New("boom"),
	}
	close(raw.successes)
	close(raw.errors)

	require.NoError(t, <-successResult.done)
	require.EqualError(t, <-errorResult.done, "kafka: publish message: boom")
	<-producer.done

	require.NotPanics(t, func() {
		completeDelivery(struct{}{}, nil)
	})
}

func TestConsumerStartShutdownAndHandler(t *testing.T) {
	t.Parallel()

	t.Run("start validates state", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, ErrConsumerClosed, (*Consumer)(nil).Start(context.Background()))

		consumer := &Consumer{}
		var nilCtx context.Context
		require.Equal(t, ErrNilContext, consumer.Start(nilCtx))
	})

	t.Run("start exits on canceled context", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		group := &mockConsumerGroup{
			consumeFn: func(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
				cancel()
				return nil
			},
		}
		consumer := &Consumer{
			group:        group,
			topics:       []string{"orders"},
			handler:      &consumerGroupHandler{handler: HandlerFunc(func(ctx context.Context, message ConsumedMessage) error { return nil })},
			shutdownDone: make(chan struct{}),
		}
		require.NoError(t, consumer.Start(ctx))
	})

	t.Run("shutdown closes group", func(t *testing.T) {
		t.Parallel()
		group := &mockConsumerGroup{}
		consumer := &Consumer{
			group:        group,
			shutdownDone: make(chan struct{}),
		}
		require.NoError(t, consumer.Shutdown(context.Background()))
		require.NoError(t, consumer.Shutdown(context.Background()))
	})

	t.Run("consume claim marks messages", func(t *testing.T) {
		t.Parallel()
		session := &mockSession{ctx: context.Background()}
		messages := make(chan *sarama.ConsumerMessage, 1)
		messages <- &sarama.ConsumerMessage{
			Topic:     "orders",
			Partition: 1,
			Offset:    2,
			Key:       []byte("k"),
			Value:     []byte("v"),
			Headers:   []*sarama.RecordHeader{{Key: []byte("x-id"), Value: []byte("1")}},
		}
		close(messages)
		claim := &mockClaim{messages: messages}

		handler := &consumerGroupHandler{
			handler: HandlerFunc(func(ctx context.Context, message ConsumedMessage) error {
				require.Equal(t, "orders", message.Topic)
				require.Equal(t, []byte("k"), message.Key)
				return nil
			}),
		}

		require.NoError(t, handler.ConsumeClaim(session, claim))
		require.Len(t, session.marked, 1)
	})

	t.Run("consume claim returns wrapped error", func(t *testing.T) {
		t.Parallel()
		session := &mockSession{ctx: context.Background()}
		messages := make(chan *sarama.ConsumerMessage, 1)
		messages <- &sarama.ConsumerMessage{Topic: "orders", Partition: 1, Offset: 2}
		close(messages)

		handler := &consumerGroupHandler{
			handler: HandlerFunc(func(ctx context.Context, message ConsumedMessage) error {
				return errors.New("boom")
			}),
		}

		err := handler.ConsumeClaim(session, &mockClaim{messages: messages})
		require.EqualError(t, err, "kafka: handle message topic=orders partition=1 offset=2: boom")
		require.EqualError(t, handler.err(), "kafka: handle message topic=orders partition=1 offset=2: boom")
		handler.reset()
		require.NoError(t, handler.err())
	})
}
