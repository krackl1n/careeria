package txmanager

import (
	"context"
	"errors"
	"fmt"
)

type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type Beginner[T Tx] interface {
	Begin(ctx context.Context) (T, error)
}

type Manager[T Tx] struct {
	beginner Beginner[T]
}

func New[T Tx](beginner Beginner[T]) *Manager[T] {
	return &Manager[T]{
		beginner: beginner,
	}
}

type txKey struct{}

func inject[T Tx](ctx context.Context, tx T) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

func TxFromContext[T Tx](ctx context.Context) (T, bool) {
	tx, ok := ctx.Value(txKey{}).(T)
	return tx, ok
}

func (m *Manager[T]) Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if _, ok := TxFromContext[T](ctx); ok {
		return fn(ctx)
	}

	tx, err := m.beginner.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	ctx = inject(ctx, tx)

	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback(ctx)
			panic(recovered)
		}

		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				err = errors.Join(err, fmt.Errorf("rollback transaction: %w", rollbackErr))
			}

			return
		}

		if commitErr := tx.Commit(ctx); commitErr != nil {
			err = fmt.Errorf("commit transaction: %w", commitErr)
		}
	}()

	err = fn(ctx)
	return err
}
