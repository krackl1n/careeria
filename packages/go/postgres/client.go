package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Client struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate postgres config: %w", err)
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	client := &Client{
		pool: pool,
	}

	if err := client.Ping(ctx); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping postgres after create: %w", err)
	}

	return client, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.pool == nil {
		return ErrClosed
	}

	if err := c.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	return nil
}

func (c *Client) Close() error {
	if c == nil || c.pool == nil {
		return nil
	}

	c.pool.Close()
	c.pool = nil

	return nil
}

func (c *Client) Begin(ctx context.Context) (pgx.Tx, error) {
	if c == nil || c.pool == nil {
		return nil, ErrClosed
	}

	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin postgres transaction: %w", err)
	}

	return tx, nil
}

func (c *Client) Pool() *pgxpool.Pool {
	if c == nil {
		return nil
	}

	return c.pool
}
