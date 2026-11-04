package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	sharedruntime "github.com/krackl1n/careeria/runtime"
	"github.com/krackl1n/careeria/shutdown"
)

func Run(parent context.Context) (result error) {
	if parent == nil {
		return errors.New("application context is required")
	}

	ctx, cancel := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dependencies, err := buildDependencies(ctx)
	if err != nil {
		return fmt.Errorf("build dependencies: %w", err)
	}
	defer func() {
		result = errors.Join(result, dependencies.Closer.Close())
	}()

	application, err := New(dependencies)
	if err != nil {
		return fmt.Errorf("create application: %w", err)
	}

	components, err := application.Runtimes(ctx)
	if err != nil {
		return fmt.Errorf("build runtime components: %w", err)
	}

	group, err := sharedruntime.Start(components...)
	if err != nil {
		return fmt.Errorf("start runtime components: %w", err)
	}

	dependencies.Logger.Info("starting hiring service application")

	waitErr := waitForRuntime(ctx, group)
	if errors.Is(waitErr, context.Canceled) && ctx.Err() != nil {
		waitErr = nil
	}

	shutdownErr := shutdown.Run(context.Background(), dependencies.Config.Shutdown, asShutdownComponents(components)...)
	if shutdownErr != nil {
		dependencies.Logger.Error("shutdown runtime components", slog.Any("error", shutdownErr))
	}

	return errors.Join(waitErr, shutdownErr)
}

func asShutdownComponents(components []sharedruntime.Component) []shutdown.Component {
	result := make([]shutdown.Component, len(components))
	for i := range components {
		result[i] = components[i]
	}
	return result
}

func waitForRuntime(ctx context.Context, group *sharedruntime.Group) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case result, ok := <-group.Results():
			if !ok {
				return nil
			}
			if result.Err != nil {
				return fmt.Errorf("runtime %q exited: %w", result.Name, result.Err)
			}
			return nil
		}
	}
}
