package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/krackl1n/careeria/retry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDo(t *testing.T) {
	t.Parallel()

	t.Run("returns error for nil context", func(t *testing.T) {
		t.Parallel()

		_, err := retry.Do[int](nil, retry.Config[int]{}, func(context.Context) (int, error) {
			return 1, nil
		})

		require.ErrorIs(t, err, retry.ErrNilContext)
	})

	t.Run("returns error for nil operation", func(t *testing.T) {
		t.Parallel()

		_, err := retry.Do[int](context.Background(), retry.Config[int]{}, nil)

		require.ErrorIs(t, err, retry.ErrNilOperation)
	})

	t.Run("does not call operation when context is already canceled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		called := false

		_, err := retry.Do[int](ctx, retry.Config[int]{
			MaxAttempts: 3,
		}, func(context.Context) (int, error) {
			called = true
			return 0, nil
		})

		require.ErrorIs(t, err, context.Canceled)
		assert.False(t, called)
	})

	t.Run("runs successful operation once", func(t *testing.T) {
		t.Parallel()

		attempts := 0

		value, err := retry.Do(context.Background(), retry.Config[int]{
			MaxAttempts: 3,
		}, func(context.Context) (int, error) {
			attempts++
			return 42, nil
		})

		require.NoError(t, err)
		assert.Equal(t, 42, value)
		assert.Equal(t, 1, attempts)
	})

	t.Run("retries errors until success", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		firstErr := errors.New("temporary")

		value, err := retry.Do(context.Background(), retry.Config[int]{
			MaxAttempts: 3,
		}, func(context.Context) (int, error) {
			attempts++
			if attempts < 3 {
				return 0, firstErr
			}

			return 7, nil
		})

		require.NoError(t, err)
		assert.Equal(t, 7, value)
		assert.Equal(t, 3, attempts)
	})

	t.Run("stops on max attempts", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		expectedErr := errors.New("still failing")

		_, err := retry.Do[int](context.Background(), retry.Config[int]{
			MaxAttempts: 2,
		}, func(context.Context) (int, error) {
			attempts++
			return attempts, expectedErr
		})

		require.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 2, attempts)
	})

	t.Run("retries by result predicate", func(t *testing.T) {
		t.Parallel()

		attempts := 0

		value, err := retry.Do(context.Background(), retry.Config[int]{
			MaxAttempts: 4,
			RetryIf: retry.OnResult(func(value int) bool {
				return value < 3
			}),
		}, func(context.Context) (int, error) {
			attempts++
			return attempts, nil
		})

		require.NoError(t, err)
		assert.Equal(t, 3, value)
		assert.Equal(t, 3, attempts)
	})

	t.Run("calls on retry with event", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("temporary")
		var events []retry.Event[int]

		_, err := retry.Do[int](context.Background(), retry.Config[int]{
			MaxAttempts: 2,
			Backoff:     retry.Constant(10 * time.Millisecond),
			OnRetry: func(event retry.Event[int]) {
				events = append(events, event)
			},
		}, func(context.Context) (int, error) {
			return 5, expectedErr
		})

		require.ErrorIs(t, err, expectedErr)
		require.Len(t, events, 1)
		assert.Equal(t, 1, events[0].Attempt)
		assert.Equal(t, 2, events[0].NextAttempt)
		assert.Equal(t, 5, events[0].Result)
		assert.ErrorIs(t, events[0].Err, expectedErr)
		assert.Equal(t, 10*time.Millisecond, events[0].Delay)
	})

	t.Run("stops when context is canceled before backoff", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		expectedErr := errors.New("temporary")

		_, err := retry.Do[int](ctx, retry.Config[int]{
			MaxAttempts: 3,
			Backoff:     retry.Constant(time.Hour),
			OnRetry: func(retry.Event[int]) {
				cancel()
			},
		}, func(context.Context) (int, error) {
			return 0, expectedErr
		})

		require.ErrorIs(t, err, expectedErr)
		require.ErrorIs(t, err, context.Canceled)
	})
}

func TestDoError(t *testing.T) {
	t.Parallel()

	attempts := 0

	err := retry.DoError(context.Background(), retry.Config[struct{}]{
		MaxAttempts: 2,
	}, func(context.Context) error {
		attempts++
		if attempts == 1 {
			return errors.New("temporary")
		}

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

func TestPredicates(t *testing.T) {
	t.Parallel()

	assert.True(t, retry.OnError[int]()(0, errors.New("boom")))
	assert.False(t, retry.OnError[int]()(0, nil))
	assert.True(t, retry.OnResult(func(value int) bool { return value == 10 })(10, nil))
	assert.False(t, retry.OnResult(func(value int) bool { return value == 10 })(10, errors.New("boom")))
	assert.False(t, retry.Never[int]()(0, errors.New("boom")))

	any := retry.Any[int](
		retry.Never[int](),
		retry.OnResult(func(value int) bool { return value == 3 }),
	)
	assert.True(t, any(3, nil))
	assert.False(t, any(4, nil))

	all := retry.All[int](
		retry.OnResult(func(value int) bool { return value > 1 }),
		retry.OnResult(func(value int) bool { return value < 3 }),
	)
	assert.True(t, all(2, nil))
	assert.False(t, all(3, nil))
	assert.False(t, retry.All[int]()(1, nil))
}

func TestBackoff(t *testing.T) {
	t.Parallel()

	assert.Equal(t, time.Duration(0), retry.NoBackoff(1))
	assert.Equal(t, time.Duration(0), retry.Constant(-time.Second)(1))
	assert.Equal(t, 5*time.Millisecond, retry.Constant(5*time.Millisecond)(3))

	backoff := retry.Exponential(100*time.Millisecond, 250*time.Millisecond)
	assert.Equal(t, 100*time.Millisecond, backoff(1))
	assert.Equal(t, 200*time.Millisecond, backoff(2))
	assert.Equal(t, 250*time.Millisecond, backoff(3))
	assert.Equal(t, 250*time.Millisecond, backoff(30))

	jitter := retry.WithJitter(
		retry.Constant(100*time.Millisecond),
		0.5,
		func() float64 { return 0 },
	)
	assert.Equal(t, 50*time.Millisecond, jitter(1))

	jitter = retry.WithJitter(
		retry.Constant(100*time.Millisecond),
		0.5,
		func() float64 { return 2 },
	)
	assert.Equal(t, 150*time.Millisecond, jitter(1))
}
