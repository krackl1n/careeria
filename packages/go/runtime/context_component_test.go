package runtime_test

import (
	"context"
	"testing"

	"github.com/krackl1n/careeria/runtime"
	"github.com/stretchr/testify/require"
)

func TestContextComponentBindsStartContext(t *testing.T) {
	type contextKey string
	startCtx := context.WithValue(t.Context(), contextKey("key"), "value")
	shutdownCtx := context.Background()
	runner := &contextRunnerStub{name: "consumer"}
	component, err := runtime.NewContextComponent(startCtx, runner)
	require.NoError(t, err)

	require.NoError(t, component.Start())
	require.NoError(t, component.Shutdown(shutdownCtx))
	require.Equal(t, "consumer", component.Name())
	require.Equal(t, startCtx, runner.startCtx)
	require.Equal(t, shutdownCtx, runner.shutdownCtx)
}

func TestContextComponentValidatesInput(t *testing.T) {
	component, err := runtime.NewContextComponent(t.Context(), &contextRunnerStub{name: " "})
	require.ErrorIs(t, err, runtime.ErrEmptyComponentName)
	require.Nil(t, component)

	component, err = runtime.NewContextComponent(nil, &contextRunnerStub{name: "consumer"})
	require.ErrorIs(t, err, runtime.ErrNilContext)
	require.Nil(t, component)

	component, err = runtime.NewContextComponent(t.Context(), (*contextRunnerStub)(nil))
	require.ErrorIs(t, err, runtime.ErrNilContextRunner)
	require.Nil(t, component)
}

type contextRunnerStub struct {
	name        string
	startCtx    context.Context
	shutdownCtx context.Context
}

func (s *contextRunnerStub) Name() string { return s.name }

func (s *contextRunnerStub) Start(ctx context.Context) error {
	s.startCtx = ctx
	return nil
}

func (s *contextRunnerStub) Shutdown(ctx context.Context) error {
	s.shutdownCtx = ctx
	return nil
}
