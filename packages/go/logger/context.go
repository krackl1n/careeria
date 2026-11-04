package logger

import (
	"context"
	"log/slog"
)

type contextKey struct{}

func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	if logger == nil {
		return ctx
	}

	return context.WithValue(ctx, contextKey{}, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}

	logger, ok := ctx.Value(contextKey{}).(*slog.Logger)
	if !ok || logger == nil {
		return slog.Default()
	}

	return logger
}
