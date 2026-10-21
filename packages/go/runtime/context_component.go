package runtime

import (
	"context"
	"reflect"
	"strings"
)

type ContextRunner interface {
	Name() string
	Start(context.Context) error
	Shutdown(context.Context) error
}

type contextComponent struct {
	name     string
	startCtx context.Context
	runner   ContextRunner
}

func NewContextComponent(startCtx context.Context, runner ContextRunner) (Component, error) {
	if startCtx == nil {
		return nil, ErrNilContext
	}
	if isNilContextRunner(runner) {
		return nil, ErrNilContextRunner
	}
	name := strings.TrimSpace(runner.Name())
	if name == "" {
		return nil, ErrEmptyComponentName
	}
	return &contextComponent{name: name, startCtx: startCtx, runner: runner}, nil
}

func (c *contextComponent) Name() string { return c.name }

func (c *contextComponent) Start() error {
	return c.runner.Start(c.startCtx)
}

func (c *contextComponent) Shutdown(ctx context.Context) error {
	return c.runner.Shutdown(ctx)
}

func isNilContextRunner(runner ContextRunner) bool {
	if runner == nil {
		return true
	}
	value := reflect.ValueOf(runner)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
