package logger

import (
	"fmt"
	"log/slog"
	"strings"
)

type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

type Config struct {
	Level     string
	Format    Format
	AddSource bool

	Service string
	Env     string
	Version string
}

func (c Config) Validate() error {
	switch c.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("logger level must be debug, info, warn, or error")
	}

	switch c.Format {
	case FormatJSON, FormatText:
	default:
		return fmt.Errorf("logger format must be %q or %q", FormatJSON, FormatText)
	}

	if strings.TrimSpace(c.Service) == "" {
		return fmt.Errorf("logger service is required")
	}

	return nil
}

func (c Config) level() slog.Level {
	switch c.Level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
