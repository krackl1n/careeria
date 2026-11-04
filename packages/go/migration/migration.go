package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const migrationsDirectory = "migrations"

func Run(
	ctx context.Context,
	logger *slog.Logger,
	cfg Config,
	command Command,
) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate migration config: %w", err)
	}

	if err := command.Validate(); err != nil {
		return err
	}

	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	migrationsFS, err := fs.Sub(cfg.FS, migrationsDirectory)
	if err != nil {
		return fmt.Errorf(
			"open migrations directory %q: %w",
			migrationsDirectory,
			err,
		)
	}

	database, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return fmt.Errorf("open migration database: %w", err)
	}
	defer database.Close()

	database.SetMaxOpenConns(2)
	database.SetMaxIdleConns(2)
	database.SetConnMaxLifetime(0)
	database.SetConnMaxIdleTime(0)

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		database,
		migrationsFS,
		goose.WithSlog(logger),
	)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	if err := provider.Ping(ctx); err != nil {
		return fmt.Errorf("ping migration database: %w", err)
	}

	logger.Info(
		"migration command started",
		slog.String("service", cfg.ServiceName),
		slog.String("command", string(command)),
	)

	switch command {
	case CommandUp:
		err = runUp(ctx, database, provider, logger, cfg.ServiceName)

	case CommandStatus:
		err = runStatus(ctx, provider, logger)

	case CommandVersion:
		err = runVersion(ctx, provider, logger)

	default:
		panic("unreachable")
	}

	if err != nil {
		return fmt.Errorf(
			"execute migration command %q: %w",
			command,
			err,
		)
	}

	logger.Info(
		"migration command completed",
		slog.String("service", cfg.ServiceName),
		slog.String("command", string(command)),
	)

	return nil
}

func runUp(
	ctx context.Context,
	database *sql.DB,
	provider *goose.Provider,
	logger *slog.Logger,
	serviceName string,
) (runErr error) {
	conn, err := database.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire advisory lock connection: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			runErr = errors.Join(
				runErr,
				fmt.Errorf("close advisory lock connection: %w", err),
			)
		}
	}()

	lockID := advisoryLockID(serviceName)

	if err := acquireAdvisoryLock(ctx, conn, lockID); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}

	defer func() {
		unlockCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := releaseAdvisoryLock(unlockCtx, conn, lockID); err != nil {
			runErr = errors.Join(
				runErr,
				fmt.Errorf("release advisory lock: %w", err),
			)
		}
	}()

	currentVersion, err := provider.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("get current migration version: %w", err)
	}

	logger.Info(
		"applying migrations",
		slog.Int64("current_version", currentVersion),
	)

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf(
			"apply migrations from version %d: %w",
			currentVersion,
			err,
		)
	}

	for _, result := range results {
		logger.Info(
			"migration applied",
			slog.String("result", result.String()),
		)
	}

	newVersion, err := provider.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("get resulting migration version: %w", err)
	}

	logger.Info(
		"migrations applied",
		slog.Int("applied_count", len(results)),
		slog.Int64("version", newVersion),
	)

	return nil
}

func runStatus(
	ctx context.Context,
	provider *goose.Provider,
	logger *slog.Logger,
) error {
	statuses, err := provider.Status(ctx)
	if err != nil {
		return fmt.Errorf("get migration status: %w", err)
	}

	for _, status := range statuses {
		attributes := []any{
			slog.Any("source", status.Source),
			slog.Any("state", status.State),
		}

		if !status.AppliedAt.IsZero() {
			attributes = append(
				attributes,
				slog.Time("applied_at", status.AppliedAt),
			)
		}

		logger.Info("migration status", attributes...)
	}

	logger.Info(
		"migration status completed",
		slog.Int("migration_count", len(statuses)),
	)

	return nil
}

func runVersion(
	ctx context.Context,
	provider *goose.Provider,
	logger *slog.Logger,
) error {
	version, err := provider.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("get migration version: %w", err)
	}

	logger.Info(
		"database migration version",
		slog.Int64("version", version),
	)

	return nil
}
