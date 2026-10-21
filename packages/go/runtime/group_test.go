package runtime_test

import (
	"context"
	"github.com/krackl1n/careeria/runtime"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStart(t *testing.T) {
	t.Parallel()

	startErr := errors.New("start failed")

	tests := []struct {
		name       string
		components []runtime.Component
		targetErr  error
	}{
		{
			name: "starts components",
			components: []runtime.Component{
				newComponent("http", nil),
				newComponent("grpc", nil),
			},
		},
		{
			name: "publishes start error",
			components: []runtime.Component{
				newComponent("consumer", startErr),
			},
		},
		{
			name:      "rejects empty list",
			targetErr: runtime.ErrNoComponents,
		},
		{
			name: "rejects nil component",
			components: []runtime.Component{
				nil,
			},
			targetErr: runtime.ErrNilComponent,
		},
		{
			name: "rejects typed nil component",
			components: []runtime.Component{
				(*componentStub)(nil),
			},
			targetErr: runtime.ErrNilComponent,
		},
		{
			name: "rejects empty name",
			components: []runtime.Component{
				newComponent(" ", nil),
			},
			targetErr: runtime.ErrEmptyComponentName,
		},
		{
			name: "rejects duplicate names",
			components: []runtime.Component{
				newComponent("http", nil),
				newComponent("http", nil),
			},
			targetErr: runtime.ErrDuplicateComponentName,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			group, err := runtime.Start(tt.components...)

			if tt.targetErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.targetErr)
				assert.Nil(t, group)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, group)

			results := make(map[string]error)

			for result := range group.Results() {
				results[result.Name] = result.Err
			}

			require.Len(t, results, len(tt.components))

			if tt.name == "publishes start error" {
				assert.ErrorIs(t, results["consumer"], startErr)
			}
		})
	}
}

func TestStartRunsConcurrently(t *testing.T) {
	t.Parallel()

	started := make(chan string, 2)
	release := make(chan struct{})

	first := &componentStub{
		name: "first",
		start: func() error {
			started <- "first"
			<-release
			return nil
		},
	}

	second := &componentStub{
		name: "second",
		start: func() error {
			started <- "second"
			<-release
			return nil
		},
	}

	group, err := runtime.Start(first, second)
	require.NoError(t, err)

	names := make(map[string]struct{}, 2)

	for len(names) < 2 {
		select {
		case name := <-started:
			names[name] = struct{}{}

		case <-time.After(time.Second):
			t.Fatal("components were not started concurrently")
		}
	}

	close(release)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()

	require.NoError(t, group.Wait(ctx))
	assert.Contains(t, names, "first")
	assert.Contains(t, names, "second")
}

func TestGroupWait(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		group     func(t *testing.T) *runtime.Group
		ctx       context.Context
		targetErr error
	}{
		{
			name: "waits for completion",
			group: func(t *testing.T) *runtime.Group {
				t.Helper()

				group, err := runtime.Start(
					newComponent("http", nil),
				)
				require.NoError(t, err)

				return group
			},
			ctx: context.Background(),
		},
		{
			name: "returns canceled context",
			group: func(t *testing.T) *runtime.Group {
				t.Helper()

				group, err := runtime.Start(
					newBlockingComponent("worker"),
				)
				require.NoError(t, err)

				return group
			},
			ctx:       canceledContext(),
			targetErr: context.Canceled,
		},
		{
			name: "returns deadline exceeded",
			group: func(t *testing.T) *runtime.Group {
				t.Helper()

				group, err := runtime.Start(
					newBlockingComponent("consumer"),
				)
				require.NoError(t, err)

				return group
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(
					context.Background(),
					time.Millisecond,
				)
				t.Cleanup(cancel)

				return ctx
			}(),
			targetErr: context.DeadlineExceeded,
		},
		{
			name: "accepts nil group",
			group: func(t *testing.T) *runtime.Group {
				t.Helper()
				return nil
			},
			ctx: context.Background(),
		},
		{
			name: "rejects nil context",
			group: func(t *testing.T) *runtime.Group {
				t.Helper()

				group, err := runtime.Start(
					newComponent("http", nil),
				)
				require.NoError(t, err)

				return group
			},
			targetErr: runtime.ErrNilContext,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.group(t).Wait(tt.ctx)

			if tt.targetErr == nil {
				require.NoError(t, err)
				return
			}

			assert.ErrorIs(t, err, tt.targetErr)
		})
	}
}

func TestGroupDone(t *testing.T) {
	t.Parallel()

	group, err := runtime.Start(
		newComponent("http", nil),
	)
	require.NoError(t, err)

	select {
	case <-group.Done():

	case <-time.After(time.Second):
		t.Fatal("group did not complete")
	}
}

type componentStub struct {
	name     string
	start    func() error
	shutdown func(context.Context) error
}

func newComponent(name string, startErr error) *componentStub {
	return &componentStub{
		name: name,
		start: func() error {
			return startErr
		},
	}
}

func newBlockingComponent(name string) *componentStub {
	return &componentStub{
		name: name,
		start: func() error {
			select {}
		},
	}
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return ctx
}

func (c *componentStub) Name() string {
	return c.name
}

func (c *componentStub) Start() error {
	return c.start()
}

func (c *componentStub) Shutdown(ctx context.Context) error {
	if c.shutdown == nil {
		return nil
	}

	return c.shutdown(ctx)
}
