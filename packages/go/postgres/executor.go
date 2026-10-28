package postgres

import (
	"context"

	"github.com/krackl1n/careeria/txmanager"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Executor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults
}

type ExecutorProvider interface {
	Executor(context.Context) Executor
}

func (c *Client) Executor(ctx context.Context) Executor {
	if c == nil || c.pool == nil {
		return nil
	}

	if tx, ok := txmanager.TxFromContext[pgx.Tx](ctx); ok && tx != nil {
		return tx
	}

	return c.pool
}
