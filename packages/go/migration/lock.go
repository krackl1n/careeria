package migration

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
)

func advisoryLockID(serviceName string) int64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(serviceName))

	return int64(hash.Sum64() & 0x7fffffffffffffff)
}

func acquireAdvisoryLock(
	ctx context.Context,
	conn *sql.Conn,
	lockID int64,
) error {
	var acquired bool

	if err := conn.QueryRowContext(
		ctx,
		`select pg_try_advisory_lock($1)`,
		lockID,
	).Scan(&acquired); err != nil {
		return fmt.Errorf("query advisory lock: %w", err)
	}

	if !acquired {
		return fmt.Errorf(
			"another migration process holds advisory lock %d",
			lockID,
		)
	}

	return nil
}

func releaseAdvisoryLock(
	ctx context.Context,
	conn *sql.Conn,
	lockID int64,
) error {
	var released bool

	if err := conn.QueryRowContext(
		ctx,
		`select pg_advisory_unlock($1)`,
		lockID,
	).Scan(&released); err != nil {
		return fmt.Errorf("query advisory unlock: %w", err)
	}

	if !released {
		return fmt.Errorf(
			"advisory lock %d was not held by the connection",
			lockID,
		)
	}

	return nil
}
