package closer

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCloser struct {
	fn func() error
}

func (c stubCloser) Close() error {
	if c.fn == nil {
		return nil
	}

	return c.fn()
}

func TestCloser_Close(t *testing.T) {
	t.Parallel()

	firstErr := errors.New("first close error")
	secondErr := errors.New("second close error")

	tests := []struct {
		name     string
		prepare  func(*Closer)
		wantErrs []error
	}{
		{
			name: "closes registered resources",
			prepare: func(closer *Closer) {
				closer.AddFunc(func() error {
					return nil
				})

				closer.Add(stubCloser{})
			},
		},
		{
			name: "joins close errors",
			prepare: func(closer *Closer) {
				closer.AddFunc(func() error {
					return firstErr
				})

				closer.AddFunc(func() error {
					return nil
				})

				closer.AddFunc(func() error {
					return secondErr
				})
			},
			wantErrs: []error{
				firstErr,
				secondErr,
			},
		},
		{
			name:    "returns nil when empty",
			prepare: func(*Closer) {},
		},
		{
			name: "ignores nil closer",
			prepare: func(closer *Closer) {
				closer.Add(nil)
			},
		},
		{
			name: "ignores nil function",
			prepare: func(closer *Closer) {
				closer.AddFunc(nil)
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			closer := New()
			tt.prepare(closer)

			err := closer.Close()

			if len(tt.wantErrs) == 0 {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)

			for _, wantErr := range tt.wantErrs {
				assert.ErrorIs(t, err, wantErr)
			}
		})
	}
}

func TestCloser_Close_ReverseOrder(t *testing.T) {
	t.Parallel()

	closer := New()
	order := make([]int, 0, 3)

	closer.AddFunc(func() error {
		order = append(order, 1)

		return nil
	})

	closer.AddFunc(func() error {
		order = append(order, 2)

		return nil
	})

	closer.AddFunc(func() error {
		order = append(order, 3)

		return nil
	})

	require.NoError(t, closer.Close())
	assert.Equal(t, []int{3, 2, 1}, order)
}

func TestCloser_Close_ContinuesAfterError(t *testing.T) {
	t.Parallel()

	closeErr := errors.New("close error")

	closer := New()

	var calls atomic.Int64

	closer.AddFunc(func() error {
		calls.Add(1)

		return nil
	})

	closer.AddFunc(func() error {
		calls.Add(1)

		return closeErr
	})

	closer.AddFunc(func() error {
		calls.Add(1)

		return nil
	})

	err := closer.Close()

	require.ErrorIs(t, err, closeErr)
	assert.Equal(t, int64(3), calls.Load())
}

func TestCloser_Close_Idempotent(t *testing.T) {
	t.Parallel()

	closer := New()

	var calls atomic.Int64

	closer.AddFunc(func() error {
		calls.Add(1)

		return nil
	})

	require.NoError(t, closer.Close())
	require.NoError(t, closer.Close())
	require.NoError(t, closer.Close())

	assert.Equal(t, int64(1), calls.Load())
}

func TestCloser_AddAfterClose_ClosesImmediately(t *testing.T) {
	t.Parallel()

	closer := New()

	require.NoError(t, closer.Close())

	var calls atomic.Int64

	closer.AddFunc(func() error {
		calls.Add(1)

		return nil
	})

	assert.Equal(t, int64(1), calls.Load())
}

func TestCloser_AddAfterClose_IgnoresCloseError(t *testing.T) {
	t.Parallel()

	closer := New()

	require.NoError(t, closer.Close())

	var calls atomic.Int64

	closer.AddFunc(func() error {
		calls.Add(1)

		return errors.New("close error")
	})

	assert.Equal(t, int64(1), calls.Load())
}

func TestCloser_Close_DoesNotHoldLockWhileClosing(t *testing.T) {
	t.Parallel()

	closer := New()

	nestedClosed := make(chan struct{})

	closer.AddFunc(func() error {
		closer.AddFunc(func() error {
			close(nestedClosed)

			return nil
		})

		return nil
	})

	done := make(chan error, 1)

	go func() {
		done <- closer.Close()
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("close deadlocked")
	}

	select {
	case <-nestedClosed:
	case <-time.After(time.Second):
		t.Fatal("nested closer was not closed")
	}
}

func TestCloser_AddAfterClose_DoesNotHoldLockWhileClosing(t *testing.T) {
	t.Parallel()

	closer := New()

	require.NoError(t, closer.Close())

	nestedClosed := make(chan struct{})

	done := make(chan struct{})

	go func() {
		defer close(done)

		closer.AddFunc(func() error {
			closer.AddFunc(func() error {
				close(nestedClosed)

				return nil
			})

			return nil
		})
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("add after close deadlocked")
	}

	select {
	case <-nestedClosed:
	case <-time.After(time.Second):
		t.Fatal("nested closer was not closed")
	}
}

func TestCloser_ConcurrentClose(t *testing.T) {
	t.Parallel()

	const goroutines = 100

	closer := New()

	var calls atomic.Int64

	closer.AddFunc(func() error {
		calls.Add(1)

		return nil
	})

	errs := make(chan error, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			errs <- closer.Close()
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}

	assert.Equal(t, int64(1), calls.Load())
}

func TestCloser_ConcurrentAddAndClose(t *testing.T) {
	t.Parallel()

	const resourcesCount = 1000

	closer := New()
	calls := make([]atomic.Int64, resourcesCount)

	var wg sync.WaitGroup
	wg.Add(resourcesCount + 1)

	for i := range resourcesCount {
		i := i

		go func() {
			defer wg.Done()

			closer.AddFunc(func() error {
				calls[i].Add(1)

				return nil
			})
		}()
	}

	go func() {
		defer wg.Done()

		assert.NoError(t, closer.Close())
	}()

	wg.Wait()

	require.NoError(t, closer.Close())

	for i := range calls {
		assert.Equalf(
			t,
			int64(1),
			calls[i].Load(),
			"resource %d must be closed exactly once",
			i,
		)
	}
}

func TestFunc_Close(t *testing.T) {
	t.Parallel()

	closeErr := errors.New("close error")

	tests := []struct {
		name    string
		fn      Func
		wantErr error
	}{
		{
			name: "calls function",
			fn: func() error {
				return nil
			},
		},
		{
			name: "returns function error",
			fn: func() error {
				return closeErr
			},
			wantErr: closeErr,
		},
		{
			name: "nil function returns nil",
			fn:   nil,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.fn.Close()

			if tt.wantErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestFunc_ImplementsIOCloser(t *testing.T) {
	t.Parallel()

	var _ io.Closer = Func(nil)
}
