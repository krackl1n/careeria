package authorizationtransaction

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	goredis "github.com/redis/go-redis/v9"
)

func (r *RedisRepository) Create(
	ctx context.Context,
	entity *domain.AuthorizationTransaction,
) error {
	if entity == nil {
		return errors.New("create authorization transaction: entity is nil")
	}

	payload, err := encode(entity)
	if err != nil {
		return fmt.Errorf("create authorization transaction: %w", err)
	}

	err = set(ctx, r.client, key(entity.ID), payload, entity.ExpiresAt.UnixMilli(), "NX")
	if errors.Is(err, goredis.Nil) {
		return domain.ErrAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("create authorization transaction: %w", err)
	}

	return nil
}
