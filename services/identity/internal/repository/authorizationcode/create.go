package authorizationcode

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationcode"
	goredis "github.com/redis/go-redis/v9"
)

func (r *RedisRepository) Create(
	ctx context.Context,
	entity *domain.AuthorizationCode,
) error {
	if entity == nil {
		return errors.New("create authorization code: entity is nil")
	}

	payload, err := encode(entity)
	if err != nil {
		return fmt.Errorf("create authorization code: %w", err)
	}

	err = r.client.Do(
		ctx,
		"SET",
		key(entity.Hash),
		payload,
		"PXAT",
		entity.ExpiresAt.UnixMilli(),
		"NX",
	).Err()
	if errors.Is(err, goredis.Nil) {
		return domain.ErrAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("create authorization code: %w", err)
	}

	return nil
}
