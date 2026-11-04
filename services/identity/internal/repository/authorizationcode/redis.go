package authorizationcode

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

type RedisClient interface {
	Do(ctx context.Context, args ...any) *goredis.Cmd
	GetDel(ctx context.Context, key string) *goredis.StringCmd
}

var _ RedisClient = (*goredis.Client)(nil)

type RedisRepository struct {
	client RedisClient
}

func NewRedisRepository(client RedisClient) *RedisRepository {
	return &RedisRepository{client: client}
}
