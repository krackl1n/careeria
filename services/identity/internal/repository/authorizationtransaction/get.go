package authorizationtransaction

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
	goredis "github.com/redis/go-redis/v9"
)

func (r *RedisRepository) Get(
	ctx context.Context,
	id domain.ID,
) (*domain.AuthorizationTransaction, error) {
	payload, err := r.client.Get(ctx, key(id)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get authorization transaction: %w", err)
	}

	entity, err := decode(payload)
	if err != nil {
		return nil, fmt.Errorf("get authorization transaction: %w", err)
	}
	if entity.ID != id {
		return nil, errors.New("get authorization transaction: stored identifier does not match key")
	}

	return entity, nil
}
