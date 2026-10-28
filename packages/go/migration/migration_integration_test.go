//go:build integration

package migration

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	dsn := integrationDSN(t)
	schema := createTestSchema(t, dsn)
	testDSN := withSearchPath(t, dsn, schema)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := Config{
		DSN:         testDSN,
		ServiceName: "migration-test-" + schema,
		Timeout:     10 * time.Second,
		FS:          testMigrations(),
	}

	t.Run("up", func(t *testing.T) {
		err := Run(
			context.Background(),
			logger,
			cfg,
			CommandUp,
		)
		require.NoError(t, err)

		database := openTestDatabase(t, testDSN)

		var tableName string

		err = database.QueryRowContext(
			context.Background(),
			`select to_regclass('users')`,
		).Scan(&tableName)
		require.NoError(t, err)
		assert.Equal(t, "users", tableName)

		var version int64

		err = database.QueryRowContext(
			context.Background(),
			`
				select coalesce(max(version_id), 0)
				from goose_db_version
				where is_applied = true
			`,
		).Scan(&version)
		require.NoError(t, err)
		assert.Equal(t, int64(2), version)
	})

	t.Run("up is idempotent", func(t *testing.T) {
		err := Run(
			context.Background(),
			logger,
			cfg,
			CommandUp,
		)
		require.NoError(t, err)

		database := openTestDatabase(t, testDSN)

		var count int

		err = database.QueryRowContext(
			context.Background(),
			`select count(*) from users`,
		).Scan(&count)
		require.NoError(t, err)
		assert.Zero(t, count)
	})

	t.Run("status", func(t *testing.T) {
		err := Run(
			context.Background(),
			logger,
			cfg,
			CommandStatus,
		)
		require.NoError(t, err)
	})

	t.Run("version", func(t *testing.T) {
		err := Run(
			context.Background(),
			logger,
			cfg,
			CommandVersion,
		)
		require.NoError(t, err)
	})
}

func TestRunValidation(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	err := Run(
		context.Background(),
		logger,
		Config{},
		CommandUp,
	)

	require.EqualError(
		t,
		err,
		"validate migration config: dsn is required",
	)
}

func TestRunUnsupportedCommand(t *testing.T) {
	t.Parallel()

	err := Run(
		context.Background(),
		nil,
		Config{
			DSN:         "postgres://unused",
			ServiceName: "identity",
			Timeout:     time.Second,
			FS:          testMigrations(),
		},
		Command("unknown"),
	)

	require.EqualError(
		t,
		err,
		`unsupported migration command: "unknown"`,
	)
}

func TestAdvisoryLock(t *testing.T) {
	dsn := integrationDSN(t)
	database := openTestDatabase(t, dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	firstConn, err := database.Conn(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, firstConn.Close())
	})

	secondConn, err := database.Conn(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, secondConn.Close())
	})

	lockID := advisoryLockID("advisory-lock-integration-test")

	require.NoError(
		t,
		acquireAdvisoryLock(ctx, firstConn, lockID),
	)

	err = acquireAdvisoryLock(ctx, secondConn, lockID)
	require.EqualError(
		t,
		err,
		fmt.Sprintf(
			"another migration process holds advisory lock %d",
			lockID,
		),
	)

	require.NoError(
		t,
		releaseAdvisoryLock(ctx, firstConn, lockID),
	)

	require.NoError(
		t,
		acquireAdvisoryLock(ctx, secondConn, lockID),
	)

	require.NoError(
		t,
		releaseAdvisoryLock(ctx, secondConn, lockID),
	)
}

func TestRunFailsWhenAdvisoryLockIsHeld(t *testing.T) {
	dsn := integrationDSN(t)
	schema := createTestSchema(t, dsn)
	testDSN := withSearchPath(t, dsn, schema)

	database := openTestDatabase(t, testDSN)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := database.Conn(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})

	serviceName := "locked-service-" + schema
	lockID := advisoryLockID(serviceName)

	require.NoError(
		t,
		acquireAdvisoryLock(ctx, conn, lockID),
	)

	t.Cleanup(func() {
		unlockCtx, unlockCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer unlockCancel()

		require.NoError(
			t,
			releaseAdvisoryLock(unlockCtx, conn, lockID),
		)
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	err = Run(
		context.Background(),
		logger,
		Config{
			DSN:         testDSN,
			ServiceName: serviceName,
			Timeout:     10 * time.Second,
			FS:          testMigrations(),
		},
		CommandUp,
	)

	require.Error(t, err)
	assert.ErrorContains(
		t,
		err,
		"another migration process holds advisory lock",
	)
}

func TestRunTimeout(t *testing.T) {
	dsn := integrationDSN(t)
	schema := createTestSchema(t, dsn)
	testDSN := withSearchPath(t, dsn, schema)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	err := Run(
		context.Background(),
		logger,
		Config{
			DSN:         testDSN,
			ServiceName: "timeout-" + schema,
			Timeout:     time.Nanosecond,
			FS:          testMigrations(),
		},
		CommandUp,
	)

	require.Error(t, err)
	assert.ErrorContains(t, err, "context deadline exceeded")
}

func testMigrations() fstest.MapFS {
	return fstest.MapFS{
		"migrations/00001_create_users.sql": {
			Data: []byte(`
-- +goose Up
create table users (
    id bigint primary key,
    email text not null unique
);

-- +goose Down
drop table users;
`),
		},
		"migrations/00002_create_user_profiles.sql": {
			Data: []byte(`
-- +goose Up
create table user_profiles (
    user_id bigint primary key references users(id),
    display_name text not null
);

-- +goose Down
drop table user_profiles;
`),
		},
	}
}

func integrationDSN(t *testing.T) string {
	t.Helper()

	dsn := os.Getenv("MIGRATION_TEST_DSN")
	if dsn == "" {
		t.Skip("MIGRATION_TEST_DSN is not set")
	}

	return dsn
}

func openTestDatabase(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	database, err := sql.Open("pgx", dsn)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	require.NoError(t, database.PingContext(ctx))

	return database
}

func createTestSchema(t *testing.T, dsn string) string {
	t.Helper()

	database := openTestDatabase(t, dsn)

	schema := fmt.Sprintf(
		"migration_test_%d",
		time.Now().UnixNano(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := database.ExecContext(
		ctx,
		"create schema "+schema,
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cleanupCancel()

		_, cleanupErr := database.ExecContext(
			cleanupCtx,
			"drop schema if exists "+schema+" cascade",
		)
		require.NoError(t, cleanupErr)
	})

	return schema
}

func withSearchPath(
	t *testing.T,
	dsn string,
	schema string,
) string {
	t.Helper()

	parsed, err := url.Parse(dsn)
	require.NoError(t, err)

	if parsed.Scheme != "postgres" &&
		parsed.Scheme != "postgresql" {
		t.Skip("integration DSN must use postgres URL format")
	}

	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()

	return parsed.String()
}

func normalizeSQL(value string) string {
	return strings.TrimSpace(value)
}
