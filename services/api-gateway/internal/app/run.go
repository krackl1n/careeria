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
	internalconfig "github.com/krackl1n/careeria/services/api-gateway/internal/config"
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
		dependencies.Logger.Info("closing application resources")

		closeErr := dependencies.Closer.Close()
		if closeErr != nil {
			dependencies.Logger.Error("close application resources", slog.Any("error", closeErr))
		} else {
			dependencies.Logger.Info("application resources closed")
		}

		result = errors.Join(result, closeErr)
	}()

	application, err := New(dependencies)
	if err != nil {
		return fmt.Errorf("create application: %w", err)
	}

	components, err := application.Runtimes(ctx)
	if err != nil {
		return fmt.Errorf("build runtime components: %w", err)
	}

	dependencies.Logger.Info("starting application", slog.Any("components", runtimeComponentNames(components)))
	group, err := sharedruntime.Start(components...)
	if err != nil {
		return fmt.Errorf("start runtime components: %w", err)
	}

	for _, component := range components {
		attributes := []slog.Attr{
			slog.String("component", component.Name()),
		}
		if addr := runtimeComponentAddr(component.Name(), dependencies.Config); addr != "" {
			attributes = append(attributes, slog.String("addr", addr))
		}

		dependencies.Logger.LogAttrs(ctx, slog.LevelInfo, "runtime component started", attributes...)
	}

	waitErr := waitForRuntime(ctx, group)
	if errors.Is(waitErr, context.Canceled) && ctx.Err() != nil {
		waitErr = nil
	}

	dependencies.Logger.Info("shutting down runtime components")
	shutdownErr := shutdown.Run(
		context.Background(),
		dependencies.Config.Shutdown,
		asShutdownComponents(components)...,
	)

	if shutdownErr == nil {
		dependencies.Logger.Info("runtime components stopped")
	} else {
		dependencies.Logger.Error("shutdown runtime components", slog.Any("error", shutdownErr))
	}

	return errors.Join(waitErr, shutdownErr)
}

func runtimeComponentNames(components []sharedruntime.Component) []string {
	result := make([]string, 0, len(components))

	for _, component := range components {
		result = append(result, component.Name())
	}

	return result
}

func runtimeComponentAddr(name string, config internalconfig.Config) string {
	switch name {
	case "http":
		return config.HTTP.Addr
	case "metrics-server":
		return config.Metrics.Addr
	default:
		return ""
	}
}

func asShutdownComponents(
	components []sharedruntime.Component,
) []shutdown.Component {
	result := make([]shutdown.Component, len(components))

	for index, component := range components {
		result[index] = component
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
