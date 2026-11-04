package authorizationtransaction

import (
	"context"
	"encoding/json"
	"fmt"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
)

const authorizationTransactionKeyPrefix = "identity:oauth:authorization-transactions:"

func set(
	ctx context.Context,
	client RedisClient,
	key string,
	payload []byte,
	expiresAtUnixMilli int64,
	mode string,
) error {
	return client.Do(
		ctx,
		"SET",
		key,
		payload,
		"PXAT",
		expiresAtUnixMilli,
		mode,
	).Err()
}

func key(id domain.ID) string {
	return authorizationTransactionKeyPrefix + id.UUID().String()
}

func encode(entity *domain.AuthorizationTransaction) ([]byte, error) {
	payload, err := json.Marshal(toModel(entity))
	if err != nil {
		return nil, fmt.Errorf("encode authorization transaction: %w", err)
	}

	return payload, nil
}

func decode(payload []byte) (*domain.AuthorizationTransaction, error) {
	var model model
	if err := json.Unmarshal(payload, &model); err != nil {
		return nil, fmt.Errorf("decode authorization transaction: %w", err)
	}

	entity, err := model.toDomain()
	if err != nil {
		return nil, fmt.Errorf("decode authorization transaction: %w", err)
	}

	return entity, nil
}
