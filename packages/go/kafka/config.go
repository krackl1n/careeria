package kafka

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IBM/sarama"
)

const (
	defaultProducerRetryMax     = 5
	defaultProducerRetryBackoff = 100 * time.Millisecond
	defaultProducerAckTimeout   = 10 * time.Second

	defaultConsumerCommitInterval    = time.Second
	defaultConsumerSessionTimeout    = 30 * time.Second
	defaultConsumerHeartbeat         = 3 * time.Second
	defaultConsumerRebalance         = 60 * time.Second
	defaultConsumerProcessingTime    = 500 * time.Millisecond
	DefaultConsumerProcessingTimeout = 30 * time.Second
)

type ConnectionConfig struct {
	Brokers  []string
	ClientID string
	Version  string

	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	TLS  TLSConfig
	SASL SASLConfig
}

type TLSConfig struct {
	Enabled            bool
	ServerName         string
	CAFile             string
	CertFile           string
	KeyFile            string
	InsecureSkipVerify bool
}

type SASLConfig struct {
	Enabled   bool
	Mechanism SASLMechanism
	Username  string
	Password  string
}

type SASLMechanism string

const (
	SASLMechanismPlain       SASLMechanism = "plain"
	SASLMechanismSCRAMSHA256 SASLMechanism = "scram-sha-256"
	SASLMechanismSCRAMSHA512 SASLMechanism = "scram-sha-512"
)

type Compression string

const (
	CompressionNone   Compression = "none"
	CompressionGZIP   Compression = "gzip"
	CompressionSnappy Compression = "snappy"
	CompressionLZ4    Compression = "lz4"
	CompressionZSTD   Compression = "zstd"
)

type InitialOffset string

const (
	InitialOffsetOldest InitialOffset = "oldest"
	InitialOffsetNewest InitialOffset = "newest"
)

type ProducerConfig struct {
	Connection ConnectionConfig

	Compression Compression

	RetryMax     int
	RetryBackoff time.Duration
	AckTimeout   time.Duration

	MaxMessageBytes int
	FlushBytes      int
	FlushMessages   int
	FlushFrequency  time.Duration
}

type ConsumerConfig struct {
	Connection ConnectionConfig

	GroupID string
	Topics  []string

	InitialOffset InitialOffset

	CommitInterval    time.Duration
	SessionTimeout    time.Duration
	HeartbeatInterval time.Duration
	RebalanceTimeout  time.Duration
	MaxProcessingTime time.Duration

	InstanceID string
}

func (c ConnectionConfig) validate() error {
	if len(c.Brokers) == 0 {
		return errors.New("kafka: brokers are required")
	}

	for _, broker := range c.Brokers {
		if strings.TrimSpace(broker) == "" {
			return errors.New("kafka: broker must not be empty")
		}
	}

	if strings.TrimSpace(c.ClientID) == "" {
		return errors.New("kafka: client id is required")
	}

	if strings.TrimSpace(c.Version) == "" {
		return errors.New("kafka: version is required")
	}

	if c.DialTimeout < 0 {
		return errors.New("kafka: dial timeout must not be negative")
	}

	if c.ReadTimeout < 0 {
		return errors.New("kafka: read timeout must not be negative")
	}

	if c.WriteTimeout < 0 {
		return errors.New("kafka: write timeout must not be negative")
	}

	if err := c.TLS.validate(); err != nil {
		return err
	}

	return c.SASL.validate()
}

func (c ProducerConfig) validate() error {
	if err := c.Connection.validate(); err != nil {
		return err
	}

	if c.RetryMax < 0 {
		return errors.New("kafka: producer retry max must not be negative")
	}

	if c.RetryBackoff < 0 {
		return errors.New("kafka: producer retry backoff must not be negative")
	}

	if c.AckTimeout < 0 {
		return errors.New("kafka: producer ack timeout must not be negative")
	}

	if c.MaxMessageBytes < 0 {
		return errors.New("kafka: producer max message bytes must not be negative")
	}

	if c.FlushBytes < 0 {
		return errors.New("kafka: producer flush bytes must not be negative")
	}

	if c.FlushMessages < 0 {
		return errors.New("kafka: producer flush messages must not be negative")
	}

	if c.FlushFrequency < 0 {
		return errors.New("kafka: producer flush frequency must not be negative")
	}

	return nil
}

func (c ProducerConfig) Validate() error { return c.validate() }

func (c ConsumerConfig) validate() error {
	if err := c.Connection.validate(); err != nil {
		return err
	}

	if strings.TrimSpace(c.GroupID) == "" {
		return errors.New("kafka: consumer group id is required")
	}

	if len(c.Topics) == 0 {
		return errors.New("kafka: consumer topics are required")
	}

	for _, topic := range c.Topics {
		if strings.TrimSpace(topic) == "" {
			return errors.New("kafka: consumer topic must not be empty")
		}
	}

	if c.CommitInterval < 0 {
		return errors.New("kafka: commit interval must not be negative")
	}

	if c.SessionTimeout < 0 {
		return errors.New("kafka: session timeout must not be negative")
	}

	if c.HeartbeatInterval < 0 {
		return errors.New("kafka: heartbeat interval must not be negative")
	}

	if c.RebalanceTimeout < 0 {
		return errors.New("kafka: rebalance timeout must not be negative")
	}

	if c.MaxProcessingTime < 0 {
		return errors.New("kafka: max processing time must not be negative")
	}

	return nil
}

func (c ConsumerConfig) Validate() error { return c.validate() }

func (c ConnectionConfig) newSaramaConfig() (*sarama.Config, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	version, err := sarama.ParseKafkaVersion(c.Version)
	if err != nil {
		return nil, fmt.Errorf("kafka: parse version: %w", err)
	}

	config := sarama.NewConfig()
	config.ClientID = c.ClientID
	config.Version = version

	if c.DialTimeout != 0 {
		config.Net.DialTimeout = c.DialTimeout
	}

	if c.ReadTimeout != 0 {
		config.Net.ReadTimeout = c.ReadTimeout
	}

	if c.WriteTimeout != 0 {
		config.Net.WriteTimeout = c.WriteTimeout
	}

	tlsConfig, err := buildTLSConfig(c.TLS)
	if err != nil {
		return nil, err
	}

	if tlsConfig != nil {
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = tlsConfig
	}

	if err := applySASL(config, c.SASL); err != nil {
		return nil, err
	}

	return config, nil
}

func (c ProducerConfig) newSaramaConfig() (*sarama.Config, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	config, err := c.Connection.newSaramaConfig()
	if err != nil {
		return nil, err
	}

	compression, err := toSaramaCompression(c.Compression)
	if err != nil {
		return nil, err
	}

	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Idempotent = true
	config.Producer.Partitioner = sarama.NewHashPartitioner
	config.Net.MaxOpenRequests = 1

	config.Producer.Retry.Max = defaultInt(
		c.RetryMax,
		defaultProducerRetryMax,
	)

	config.Producer.Retry.Backoff = defaultDuration(
		c.RetryBackoff,
		defaultProducerRetryBackoff,
	)

	config.Producer.Timeout = defaultDuration(
		c.AckTimeout,
		defaultProducerAckTimeout,
	)

	config.Producer.Compression = compression

	if c.MaxMessageBytes != 0 {
		config.Producer.MaxMessageBytes = c.MaxMessageBytes
	}

	if c.FlushBytes != 0 {
		config.Producer.Flush.Bytes = c.FlushBytes
	}

	if c.FlushMessages != 0 {
		config.Producer.Flush.Messages = c.FlushMessages
	}

	if c.FlushFrequency != 0 {
		config.Producer.Flush.Frequency = c.FlushFrequency
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("kafka: validate producer config: %w", err)
	}

	return config, nil
}

func (c ConsumerConfig) newSaramaConfig() (*sarama.Config, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	config, err := c.Connection.newSaramaConfig()
	if err != nil {
		return nil, err
	}

	initialOffset, err := toSaramaInitialOffset(c.InitialOffset)
	if err != nil {
		return nil, err
	}

	config.Consumer.Return.Errors = false
	config.Consumer.IsolationLevel = sarama.ReadCommitted

	config.Consumer.Offsets.AutoCommit.Enable = true
	config.Consumer.Offsets.AutoCommit.Interval = defaultDuration(
		c.CommitInterval,
		defaultConsumerCommitInterval,
	)

	config.Consumer.Offsets.Initial = initialOffset

	config.Consumer.Group.Session.Timeout = defaultDuration(
		c.SessionTimeout,
		defaultConsumerSessionTimeout,
	)

	config.Consumer.Group.Heartbeat.Interval = defaultDuration(
		c.HeartbeatInterval,
		defaultConsumerHeartbeat,
	)

	config.Consumer.Group.Rebalance.Timeout = defaultDuration(
		c.RebalanceTimeout,
		defaultConsumerRebalance,
	)

	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategySticky(),
	}

	config.Consumer.Group.InstanceId = c.InstanceID

	config.Consumer.MaxProcessingTime = defaultDuration(
		c.MaxProcessingTime,
		defaultConsumerProcessingTime,
	)

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("kafka: validate consumer config: %w", err)
	}

	return config, nil
}

func toSaramaCompression(
	compression Compression,
) (sarama.CompressionCodec, error) {
	switch compression {
	case "", CompressionSnappy:
		return sarama.CompressionSnappy, nil
	case CompressionNone:
		return sarama.CompressionNone, nil
	case CompressionGZIP:
		return sarama.CompressionGZIP, nil
	case CompressionLZ4:
		return sarama.CompressionLZ4, nil
	case CompressionZSTD:
		return sarama.CompressionZSTD, nil
	default:
		return sarama.CompressionNone, fmt.Errorf(
			"kafka: unsupported compression %q",
			compression,
		)
	}
}

func toSaramaInitialOffset(
	offset InitialOffset,
) (int64, error) {
	switch offset {
	case "", InitialOffsetOldest:
		return sarama.OffsetOldest, nil
	case InitialOffsetNewest:
		return sarama.OffsetNewest, nil
	default:
		return 0, fmt.Errorf(
			"kafka: unsupported initial offset %q",
			offset,
		)
	}
}

func defaultInt(value, fallback int) int {
	if value == 0 {
		return fallback
	}

	return value
}

func defaultDuration(
	value,
	fallback time.Duration,
) time.Duration {
	if value == 0 {
		return fallback
	}

	return value
}
