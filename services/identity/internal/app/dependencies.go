package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/krackl1n/careeria/closer"
	sharedlogger "github.com/krackl1n/careeria/logger"
	"github.com/krackl1n/careeria/postgres"
	internalconfig "github.com/krackl1n/careeria/services/identity/internal/config"
	sharedtracing "github.com/krackl1n/careeria/tracing"
	goredis "github.com/redis/go-redis/v9"
)

type Dependencies struct {
	Config         internalconfig.Config
	Logger         *slog.Logger
	Closer         *closer.Closer
	PostgresClient *postgres.Client
	RedisClient    *goredis.Client
}

func buildDependencies(ctx context.Context) (*Dependencies, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}

	cl := closer.New()
	cfg, err := internalconfig.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	logger := sharedlogger.New(cfg.Logger, nil)
	sharedlogger.SetDefault(logger)
	tracing, err := sharedtracing.New(ctx, cfg.Tracing)
	if err != nil {
		return nil, fmt.Errorf("initialize tracing: %w", err)
	}
	cl.AddFunc(func() error { return tracing.Close(context.Background()) })
	client, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("initialize postgres client: %w", err)
	}
	cl.Add(client)
	options, err := goredis.ParseURL(cfg.Redis.URL)
	if err != nil {
		return nil, fmt.Errorf("parse Redis URL: %w", err)
	}
	redisClient := goredis.NewClient(options)
	cl.Add(redisClient)
	return &Dependencies{Config: cfg, Logger: logger, Closer: cl, PostgresClient: client, RedisClient: redisClient}, nil
}
