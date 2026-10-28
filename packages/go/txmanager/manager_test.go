package txmanager

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	errBegin    = errors.New("begin failed")
	errCommit   = errors.New("commit failed")
	errRollback = errors.New("rollback failed")
	errFn       = errors.New("fn failed")
)

type mockTx struct {
	commits   int
	rollbacks int

	commitErr   error
	rollbackErr error
}

func (tx *mockTx) Commit(ctx context.Context) error {
	tx.commits++
	return tx.commitErr
}

func (tx *mockTx) Rollback(ctx context.Context) error {
	tx.rollbacks++
	return tx.rollbackErr
}

type mockBeginner struct {
	tx *mockTx

	begins int
	err    error
}

func (b *mockBeginner) Begin(ctx context.Context) (*mockTx, error) {
	b.begins++

	if b.err != nil {
		return nil, b.err
	}

	return b.tx, nil
}

func TestManager_Do_Success(t *testing.T) {
	tx := &mockTx{}
	beginner := &mockBeginner{tx: tx}
	manager := New(beginner)

	err := manager.Do(context.Background(), func(ctx context.Context) error {
		gotTx, ok := TxFromContext[*mockTx](ctx)

		require.True(t, ok)
		require.Same(t, tx, gotTx)

		return nil
	})

	require.NoError(t, err)
	require.Equal(t, 1, beginner.begins)
	require.Equal(t, 1, tx.commits)
	require.Equal(t, 0, tx.rollbacks)
}

func TestManager_Do_BeginError(t *testing.T) {
	beginner := &mockBeginner{
		err: errBegin,
	}
	manager := New(beginner)

	err := manager.Do(context.Background(), func(ctx context.Context) error {
		require.Fail(t, "fn must not be called")
		return nil
	})

	require.ErrorIs(t, err, errBegin)
	require.Equal(t, 1, beginner.begins)
}

func TestManager_Do_FnErrorRollbackSuccess(t *testing.T) {
	tx := &mockTx{}
	beginner := &mockBeginner{tx: tx}
	manager := New(beginner)

	err := manager.Do(context.Background(), func(ctx context.Context) error {
		return errFn
	})

	require.ErrorIs(t, err, errFn)
	require.Equal(t, 1, beginner.begins)
	require.Equal(t, 0, tx.commits)
	require.Equal(t, 1, tx.rollbacks)
}

func TestManager_Do_FnErrorRollbackError(t *testing.T) {
	tx := &mockTx{
		rollbackErr: errRollback,
	}
	beginner := &mockBeginner{tx: tx}
	manager := New(beginner)

	err := manager.Do(context.Background(), func(ctx context.Context) error {
		return errFn
	})

	require.ErrorIs(t, err, errFn)
	require.ErrorIs(t, err, errRollback)
	require.Equal(t, 1, beginner.begins)
	require.Equal(t, 0, tx.commits)
	require.Equal(t, 1, tx.rollbacks)
}

func TestManager_Do_CommitError(t *testing.T) {
	tx := &mockTx{
		commitErr: errCommit,
	}
	beginner := &mockBeginner{tx: tx}
	manager := New(beginner)

	err := manager.Do(context.Background(), func(ctx context.Context) error {
		return nil
	})

	require.ErrorIs(t, err, errCommit)
	require.Equal(t, 1, beginner.begins)
	require.Equal(t, 1, tx.commits)
	require.Equal(t, 0, tx.rollbacks)
}

func TestManager_Do_PanicRollbackSuccess(t *testing.T) {
	tx := &mockTx{}
	beginner := &mockBeginner{tx: tx}
	manager := New(beginner)

	require.PanicsWithValue(t, "boom", func() {
		_ = manager.Do(context.Background(), func(ctx context.Context) error {
			panic("boom")
		})
	})

	require.Equal(t, 1, beginner.begins)
	require.Equal(t, 0, tx.commits)
	require.Equal(t, 1, tx.rollbacks)
}

func TestManager_Do_PanicRollbackError(t *testing.T) {
	tx := &mockTx{
		rollbackErr: errRollback,
	}
	beginner := &mockBeginner{tx: tx}
	manager := New(beginner)

	require.PanicsWithValue(t, "boom", func() {
		_ = manager.Do(context.Background(), func(ctx context.Context) error {
			panic("boom")
		})
	})

	require.Equal(t, 1, beginner.begins)
	require.Equal(t, 0, tx.commits)
	require.Equal(t, 1, tx.rollbacks)
}

func TestManager_Do_NestedTransactionSuccess(t *testing.T) {
	tx := &mockTx{}
	beginner := &mockBeginner{tx: tx}
	manager := New(beginner)

	err := manager.Do(context.Background(), func(ctx context.Context) error {
		return manager.Do(ctx, func(ctx context.Context) error {
			gotTx, ok := TxFromContext[*mockTx](ctx)

			require.True(t, ok)
			require.Same(t, tx, gotTx)

			return nil
		})
	})

	require.NoError(t, err)
	require.Equal(t, 1, beginner.begins)
	require.Equal(t, 1, tx.commits)
	require.Equal(t, 0, tx.rollbacks)
}

func TestManager_Do_NestedTransactionError(t *testing.T) {
	tx := &mockTx{}
	beginner := &mockBeginner{tx: tx}
	manager := New(beginner)

	err := manager.Do(context.Background(), func(ctx context.Context) error {
		return manager.Do(ctx, func(ctx context.Context) error {
			return errFn
		})
	})

	require.ErrorIs(t, err, errFn)
	require.Equal(t, 1, beginner.begins)
	require.Equal(t, 0, tx.commits)
	require.Equal(t, 1, tx.rollbacks)
}

func TestManager_Do_NestedTransactionPanic(t *testing.T) {
	tx := &mockTx{}
	beginner := &mockBeginner{tx: tx}
	manager := New(beginner)

	require.PanicsWithValue(t, "boom", func() {
		_ = manager.Do(context.Background(), func(ctx context.Context) error {
			return manager.Do(ctx, func(ctx context.Context) error {
				panic("boom")
			})
		})
	})

	require.Equal(t, 1, beginner.begins)
	require.Equal(t, 0, tx.commits)
	require.Equal(t, 1, tx.rollbacks)
}

func TestTxFromContext_Empty(t *testing.T) {
	tx, ok := TxFromContext[*mockTx](context.Background())

	require.False(t, ok)
	require.Nil(t, tx)
}
