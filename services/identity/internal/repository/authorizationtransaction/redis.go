package authorizationtransaction

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

type RedisClient interface {
	Do(ctx context.Context, args ...any) *goredis.Cmd
	Get(ctx context.Context, key string) *goredis.StringCmd
	Del(ctx context.Context, keys ...string) *goredis.IntCmd
}

var _ RedisClient = (*goredis.Client)(nil)

type RedisRepository struct {
	client RedisClient
}

func NewRedisRepository(client RedisClient) *RedisRepository {
	return &RedisRepository{client: client}
}
