package authorizationcode

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationcode"
	goredis "github.com/redis/go-redis/v9"
)

func (r *RedisRepository) Consume(
	ctx context.Context,
	hash domain.Hash,
) (*domain.AuthorizationCode, error) {
	payload, err := r.client.GetDel(ctx, key(hash)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consume authorization code: %w", err)
	}

	entity, err := decode(payload)
	if err != nil {
		return nil, fmt.Errorf("consume authorization code: %w", err)
	}
	if entity.Hash != hash {
		return nil, errors.New("consume authorization code: stored hash does not match key")
	}

	return entity, nil
}
