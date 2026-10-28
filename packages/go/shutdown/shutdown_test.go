package shutdown_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/krackl1n/careeria/shutdown"
)

func TestRun(t *testing.T) {
	t.Parallel()

	shutdownErr := errors.New("shutdown failed")

	tests := []struct {
		name       string
		parent     context.Context
		cfg        shutdown.Config
		components []shutdown.Component
		assert     func(t *testing.T, err error)
	}{
		{
			name:   "shuts down components",
			parent: context.Background(),
			cfg: shutdown.Config{
				Timeout: time.Second,
			},
			components: []shutdown.Component{
				newStubComponent("first", nil),
				newStubComponent("second", nil),
			},
			assert: func(t *testing.T, err error) {
				t.Helper()

				require.NoError(t, err)
			},
		},
		{
			name:   "joins component errors",
			parent: context.Background(),
			cfg: shutdown.Config{
				Timeout: time.Second,
			},
			components: []shutdown.Component{
				newStubComponent("first", shutdownErr),
				newStubComponent("second", shutdownErr),
			},
			assert: func(t *testing.T, err error) {
				t.Helper()

				require.Error(t, err)
				assert.ErrorIs(t, err, shutdownErr)
				assert.Contains(t, err.Error(), `shutdown "first"`)
				assert.Contains(t, err.Error(), `shutdown "second"`)
			},
		},
		{
			name:   "rejects nil context",
			parent: nil,
			cfg: shutdown.Config{
				Timeout: time.Second,
			},
			assert: func(t *testing.T, err error) {
				t.Helper()

				assert.ErrorIs(t, err, shutdown.ErrNilContext)
			},
		},
		{
			name:   "rejects zero timeout",
			parent: context.Background(),
			cfg: shutdown.Config{
				Timeout: 0,
			},
			assert: func(t *testing.T, err error) {
				t.Helper()

				assert.ErrorIs(t, err, shutdown.ErrInvalidTimeout)
			},
		},
		{
			name:   "rejects negative timeout",
			parent: context.Background(),
			cfg: shutdown.Config{
				Timeout: -time.Second,
			},
			assert: func(t *testing.T, err error) {
				t.Helper()

				assert.ErrorIs(t, err, shutdown.ErrInvalidTimeout)
			},
		},
		{
			name:   "rejects nil component",
			parent: context.Background(),
			cfg: shutdown.Config{
				Timeout: time.Second,
			},
			components: []shutdown.Component{
				nil,
			},
			assert: func(t *testing.T, err error) {
				t.Helper()

				assert.ErrorIs(t, err, shutdown.ErrNilComponent)
			},
		},
		{
			name:   "rejects typed nil component",
			parent: context.Background(),
			cfg: shutdown.Config{
				Timeout: time.Second,
			},
			components: []shutdown.Component{
				(*stubComponent)(nil),
			},
			assert: func(t *testing.T, err error) {
				t.Helper()

				assert.ErrorIs(t, err, shutdown.ErrNilComponent)
			},
		},
		{
			name: "handles canceled parent context",
			parent: func() context.Context {
				ctx, cancel := context.WithCancel(
					context.Background(),
				)
				cancel()

				return ctx
			}(),
			cfg: shutdown.Config{
				Timeout: time.Second,
			},
			components: []shutdown.Component{
				newStubComponent("component", nil),
			},
			assert: func(t *testing.T, err error) {
				t.Helper()

				require.Error(t, err)
				assert.ErrorIs(t, err, context.Canceled)
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := shutdown.Run(
				tt.parent,
				tt.cfg,
				tt.components...,
			)

			tt.assert(t, err)
		})
	}
}

func TestRun_ShutsDownInReverseOrder(t *testing.T) {
	t.Parallel()

	var (
		mu    sync.Mutex
		order []string
	)

	components := []shutdown.Component{
		newRecordingComponent("first", &mu, &order),
		newRecordingComponent("second", &mu, &order),
		newRecordingComponent("third", &mu, &order),
	}

	err := shutdown.Run(
		context.Background(),
		shutdown.Config{
			Timeout: time.Second,
		},
		components...,
	)

	require.NoError(t, err)

	assert.Equal(
		t,
		[]string{
			"third",
			"second",
			"first",
		},
		order,
	)
}

func TestRun_ContinuesAfterComponentError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("component failed")

	var (
		mu    sync.Mutex
		order []string
	)

	components := []shutdown.Component{
		newRecordingComponent("first", &mu, &order),
		&stubComponent{
			name: "second",
			shutdown: func(context.Context) error {
				mu.Lock()
				order = append(order, "second")
				mu.Unlock()

				return expectedErr
			},
		},
		newRecordingComponent("third", &mu, &order),
	}

	err := shutdown.Run(
		context.Background(),
		shutdown.Config{
			Timeout: time.Second,
		},
		components...,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)

	assert.Equal(
		t,
		[]string{
			"third",
			"second",
			"first",
		},
		order,
	)
}

func TestRun_StopsAfterTimeout(t *testing.T) {
	t.Parallel()

	var (
		mu    sync.Mutex
		order []string
	)

	first := newRecordingComponent(
		"first",
		&mu,
		&order,
	)

	blocking := &stubComponent{
		name: "blocking",
		shutdown: func(ctx context.Context) error {
			mu.Lock()
			order = append(order, "blocking")
			mu.Unlock()

			<-ctx.Done()

			return ctx.Err()
		},
	}

	err := shutdown.Run(
		context.Background(),
		shutdown.Config{
			Timeout: 20 * time.Millisecond,
		},
		first,
		blocking,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	assert.Equal(
		t,
		[]string{
			"blocking",
		},
		order,
	)
}

type stubComponent struct {
	name     string
	shutdown func(context.Context) error
}

func newStubComponent(
	name string,
	err error,
) *stubComponent {
	return &stubComponent{
		name: name,
		shutdown: func(context.Context) error {
			return err
		},
	}
}

func newRecordingComponent(
	name string,
	mu *sync.Mutex,
	order *[]string,
) *stubComponent {
	return &stubComponent{
		name: name,
		shutdown: func(context.Context) error {
			mu.Lock()
			defer mu.Unlock()

			*order = append(*order, name)

			return nil
		},
	}
}

func (c *stubComponent) Name() string {
	return c.name
}

func (c *stubComponent) Shutdown(
	ctx context.Context,
) error {
	return c.shutdown(ctx)
}
