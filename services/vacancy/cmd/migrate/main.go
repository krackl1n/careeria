package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	sharedlogger "github.com/krackl1n/careeria/logger"
	"github.com/krackl1n/careeria/migration"
	"github.com/krackl1n/careeria/services/vacancy/db"
	internalconfig "github.com/krackl1n/careeria/services/vacancy/internal/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("migration binary failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	args := os.Args[1:]
	command := migration.CommandUp
	if len(args) > 1 {
		return fmt.Errorf("usage: migrate [up|status|version]")
	}
	if len(args) == 1 {
		switch args[0] {
		case "help", "-h", "--help":
			fmt.Println("Usage: migrate [up|status|version]")
			return nil
		}
		parsed, err := migration.ParseCommand(args[0])
		if err != nil {
			return err
		}
		command = parsed
	}

	cfg, err := internalconfig.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := sharedlogger.New(cfg.Logger, nil)
	sharedlogger.SetDefault(logger)

	if err := migration.Run(ctx, logger, migration.Config{
		DSN:         cfg.Postgres.DSN(),
		ServiceName: "vacancy",
		Timeout:     time.Minute,
		FS:          db.Migrations(),
	}, command); err != nil {
		return fmt.Errorf("run migration command: %w", err)
	}

	return nil
}
