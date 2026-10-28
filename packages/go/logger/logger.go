package logger

import (
	"io"
	"log/slog"
	"os"
)

func New(cfg Config, out io.Writer) *slog.Logger {
	if out == nil {
		out = os.Stdout
	}

	opts := &slog.HandlerOptions{
		Level:     cfg.level(),
		AddSource: cfg.AddSource,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			switch attr.Key {
			case slog.TimeKey:
				attr.Key = "timestamp"
			case slog.LevelKey:
				attr.Key = "severity"
			case slog.MessageKey:
				attr.Key = "message"
			}

			return attr
		},
	}

	var handler slog.Handler

	switch cfg.Format {
	case FormatText:
		handler = slog.NewTextHandler(out, opts)
	default:
		handler = slog.NewJSONHandler(out, opts)
	}

	logger := slog.New(handler)

	attrs := make([]any, 0, 3)

	if cfg.Service != "" {
		attrs = append(attrs, slog.String("service", cfg.Service))
	}
	if cfg.Env != "" {
		attrs = append(attrs, slog.String("env", cfg.Env))
	}
	if cfg.Version != "" {
		attrs = append(attrs, slog.String("version", cfg.Version))
	}

	if len(attrs) > 0 {
		logger = logger.With(attrs...)
	}

	return logger
}

func SetDefault(logger *slog.Logger) {
	if logger != nil {
		slog.SetDefault(logger)
	}
}
