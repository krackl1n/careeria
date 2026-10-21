package authorizationtransaction

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	goredis "github.com/redis/go-redis/v9"
)

func (r *RedisRepository) Update(
	ctx context.Context,
	entity *domain.AuthorizationTransaction,
) error {
	if entity == nil {
		return errors.New("update authorization transaction: entity is nil")
	}

	payload, err := encode(entity)
	if err != nil {
		return fmt.Errorf("update authorization transaction: %w", err)
	}

	err = set(ctx, r.client, key(entity.ID), payload, entity.ExpiresAt.UnixMilli(), "XX")
	if errors.Is(err, goredis.Nil) {
		return domain.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("update authorization transaction: %w", err)
	}

	return nil
}
