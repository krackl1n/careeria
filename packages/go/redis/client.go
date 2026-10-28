package redis

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

type Client struct {
	client *goredis.Client
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate redis config: %w", err)
	}

	tlsConfig, err := buildTLSConfig(cfg.TLS)
	if err != nil {
		return nil, fmt.Errorf("build redis tls config: %w", err)
	}

	redisClient := goredis.NewClient(&goredis.Options{
		Addr:      cfg.Connection.Address(),
		Username:  cfg.Connection.Username,
		Password:  cfg.Connection.Password,
		DB:        cfg.Connection.Database,
		TLSConfig: tlsConfig,

		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,

		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		PoolTimeout:     cfg.PoolTimeout,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		ConnMaxLifetime: cfg.ConnMaxLifetime,

		MaxRetries:      cfg.MaxRetries,
		MinRetryBackoff: cfg.MinRetryBackoff,
		MaxRetryBackoff: cfg.MaxRetryBackoff,
	})

	client := &Client{
		client: redisClient,
	}

	if err := client.Ping(ctx); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis after create: %w", err)
	}

	return client, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.client == nil {
		return ErrClosed
	}

	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}

	return nil
}

func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}

	if err := c.client.Close(); err != nil {
		return fmt.Errorf("close redis client: %w", err)
	}

	c.client = nil

	return nil
}

func (c *Client) Client() *goredis.Client {
	if c == nil {
		return nil
	}

	return c.client
}

func (c *Client) Stats() *goredis.PoolStats {
	if c == nil || c.client == nil {
		return nil
	}

	return c.client.PoolStats()
}
