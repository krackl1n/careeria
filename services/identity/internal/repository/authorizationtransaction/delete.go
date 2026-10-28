package authorizationtransaction

import (
	"context"
	"fmt"

	domain "github.com/krackl1n/careeria/services/identity/internal/domain/authorizationtransaction"
)

func (r *RedisRepository) Delete(ctx context.Context, id domain.ID) error {
	deleted, err := r.client.Del(ctx, key(id)).Result()
	if err != nil {
		return fmt.Errorf("delete authorization transaction: %w", err)
	}
	if deleted == 0 {
		return domain.ErrNotFound
	}

	return nil
}
