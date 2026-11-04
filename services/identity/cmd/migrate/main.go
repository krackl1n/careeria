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
	"github.com/krackl1n/careeria/services/identity/db"
	internalconfig "github.com/krackl1n/careeria/services/identity/internal/config"
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

	command, err := parseCommand(os.Args[1:])
	if err != nil {
		return err
	}

	cfg, err := internalconfig.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := sharedlogger.New(cfg.Logger, nil)
	sharedlogger.SetDefault(logger)

	if err := migration.Run(ctx, logger, migration.Config{
		DSN:         cfg.Postgres.DSN(),
		ServiceName: cfg.Logger.Service,
		Timeout:     time.Minute,
		FS:          db.Migrations(),
	}, command); err != nil {
		return fmt.Errorf("run migration command: %w", err)
	}

	return nil
}

func parseCommand(args []string) (migration.Command, error) {
	if len(args) == 0 {
		return migration.CommandUp, nil
	}

	switch args[0] {
	case string(migration.CommandUp):
		return migration.CommandUp, nil
	case string(migration.CommandStatus):
		return migration.CommandStatus, nil
	case string(migration.CommandVersion):
		return migration.CommandVersion, nil
	case "-h", "--help", "help":
		printUsage()
		os.Exit(0)
		return "", nil
	default:
		return "", fmt.Errorf("unsupported migration command: %s", args[0])
	}
}

func printUsage() {
	_, _ = os.Stderr.WriteString(
		"Usage:\n" +
			"  migrate up\n" +
			"  migrate status\n" +
			"  migrate version\n",
	)
}
