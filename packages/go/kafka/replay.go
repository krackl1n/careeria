package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
)

// ResetConsumerGroupOffsets removes committed offsets so the next consumer
// session starts according to ConsumerConfig.InitialOffset. The group must be
// stopped while this operation runs.
func ResetConsumerGroupOffsets(ctx context.Context, cfg ConsumerConfig) error {
	if ctx == nil {
		return ErrNilContext
	}
	if err := cfg.validate(); err != nil {
		return err
	}
	saramaConfig, err := cfg.Connection.newSaramaConfig()
	if err != nil {
		return err
	}
	client, err := sarama.NewClient(cfg.Connection.Brokers, saramaConfig)
	if err != nil {
		return fmt.Errorf("kafka: create replay admin client: %w", err)
	}
	defer client.Close()
	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		return fmt.Errorf("kafka: create replay admin: %w", err)
	}
	defer admin.Close()
	for _, topic := range cfg.Topics {
		if err := ctx.Err(); err != nil {
			return err
		}
		partitions, err := client.Partitions(topic)
		if err != nil {
			return fmt.Errorf("kafka: list partitions for %s: %w", topic, err)
		}
		for _, partition := range partitions {
			if err := admin.DeleteConsumerGroupOffset(cfg.GroupID, topic, partition); err != nil {
				return fmt.Errorf("kafka: reset %s[%d] for group %s: %w", topic, partition, cfg.GroupID, err)
			}
		}
	}
	return nil
}
