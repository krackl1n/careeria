package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/krackl1n/careeria/closer"
	sharedlogger "github.com/krackl1n/careeria/logger"
	"github.com/krackl1n/careeria/postgres"
	internalconfig "github.com/krackl1n/careeria/services/hiring/internal/config"
	sharedtracing "github.com/krackl1n/careeria/tracing"
)

type Dependencies struct {
	Config         internalconfig.Config
	Logger         *slog.Logger
	Closer         *closer.Closer
	PostgresClient *postgres.Client
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
	cl.AddFunc(func() error {
		return tracing.Close(context.Background())
	})

	client, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("initialize postgres client: %w", err)
	}
	cl.Add(client)

	return &Dependencies{
		Config:         cfg,
		Logger:         logger,
		Closer:         cl,
		PostgresClient: client,
	}, nil
}

