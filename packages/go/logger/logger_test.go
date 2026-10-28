package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew_JSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  Config
		want map[string]any
	}{
		{
			name: "default json logger",
			cfg: Config{
				Format: FormatJSON,
			},
			want: map[string]any{
				"severity": "INFO",
				"message":  "test message",
			},
		},
		{
			name: "with service env version",
			cfg: Config{
				Format:  FormatJSON,
				Service: "users",
				Env:     "production",
				Version: "1.2.3",
			},
			want: map[string]any{
				"severity": "INFO",
				"message":  "test message",
				"service":  "users",
				"env":      "production",
				"version":  "1.2.3",
			},
		},
		{
			name: "with source",
			cfg: Config{
				Format:    FormatJSON,
				AddSource: true,
			},
			want: map[string]any{
				"severity": "INFO",
				"message":  "test message",
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			log := New(tt.cfg, &buf)
			log.Info("test message")

			var got map[string]any
			require.NoError(t, json.Unmarshal(buf.Bytes(), &got))

			for key, value := range tt.want {
				require.Equal(t, value, got[key])
			}

			require.Contains(t, got, "timestamp")

			if tt.cfg.AddSource {
				require.Contains(t, got, "source")
			}
		})
	}
}

func TestNew_Text(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  Config
		want []string
	}{
		{
			name: "text format",
			cfg: Config{
				Format: FormatText,
			},
			want: []string{
				"severity=INFO",
				`message="test message"`,
			},
		},
		{
			name: "text format with service env version",
			cfg: Config{
				Format:  FormatText,
				Service: "billing",
				Env:     "local",
				Version: "dev",
			},
			want: []string{
				"severity=INFO",
				`message="test message"`,
				"service=billing",
				"env=local",
				"version=dev",
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			log := New(tt.cfg, &buf)
			log.Info("test message")

			got := buf.String()

			for _, want := range tt.want {
				require.Contains(t, got, want)
			}
		})
	}
}

func TestNew_LevelFiltering(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		level     string
		log       func(*slog.Logger)
		wantEmpty bool
	}{
		{
			name:  "debug level allows debug",
			level: "debug",
			log: func(log *slog.Logger) {
				log.Debug("debug message")
			},
			wantEmpty: false,
		},
		{
			name:  "info level filters debug",
			level: "info",
			log: func(log *slog.Logger) {
				log.Debug("debug message")
			},
			wantEmpty: true,
		},
		{
			name:  "warn level filters info",
			level: "warn",
			log: func(log *slog.Logger) {
				log.Info("info message")
			},
			wantEmpty: true,
		},
		{
			name:  "warn level allows warn",
			level: "warn",
			log: func(log *slog.Logger) {
				log.Warn("warn message")
			},
			wantEmpty: false,
		},
		{
			name:  "error level filters warn",
			level: "error",
			log: func(log *slog.Logger) {
				log.Warn("warn message")
			},
			wantEmpty: true,
		},
		{
			name:  "error level allows error",
			level: "error",
			log: func(log *slog.Logger) {
				log.Error("error message")
			},
			wantEmpty: false,
		},
		{
			name:  "unknown level defaults to info and filters debug",
			level: "unknown",
			log: func(log *slog.Logger) {
				log.Debug("debug message")
			},
			wantEmpty: true,
		},
		{
			name:  "empty level defaults to info and allows info",
			level: "",
			log: func(log *slog.Logger) {
				log.Info("info message")
			},
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			log := New(Config{
				Level:  tt.level,
				Format: FormatJSON,
			}, &buf)

			tt.log(log)

			if tt.wantEmpty {
				require.Empty(t, buf.String())
				return
			}

			require.NotEmpty(t, buf.String())
		})
	}
}

func TestSetDefault(t *testing.T) {
	t.Parallel()

	t.Run("nil logger does nothing", func(t *testing.T) {
		t.Parallel()

		require.NotPanics(t, func() {
			SetDefault(nil)
		})
	})

	t.Run("sets default logger", func(t *testing.T) {
		var buf bytes.Buffer

		previous := slog.Default()
		t.Cleanup(func() {
			slog.SetDefault(previous)
		})

		log := New(Config{
			Format: FormatJSON,
		}, &buf)

		SetDefault(log)

		slog.Info("default message")

		var got map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
		require.Equal(t, "default message", got["message"])
	})
}
