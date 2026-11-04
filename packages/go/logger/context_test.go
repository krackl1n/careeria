package logger

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		log  *slog.Logger
		want bool
	}{
		{
			name: "logger exists",
			log:  slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)),
			want: true,
		},
		{
			name: "logger is nil",
			log:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			got := WithContext(ctx, tt.log)

			if tt.want {
				require.NotEqual(t, ctx, got)
				require.Same(t, tt.log, FromContext(got))
				return
			}

			require.Equal(t, ctx, got)
		})
	}
}

func TestFromContext(t *testing.T) {
	t.Parallel()

	custom := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))

	tests := []struct {
		name string
		ctx  context.Context
		want *slog.Logger
	}{
		{
			name: "logger from context",
			ctx:  WithContext(context.Background(), custom),
			want: custom,
		},
		{
			name: "empty context returns default logger",
			ctx:  context.Background(),
			want: nil,
		},
		{
			name: "nil context returns default logger",
			ctx:  nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := FromContext(tt.ctx)

			if tt.want == nil {
				require.NotNil(t, got)
				return
			}

			require.Same(t, tt.want, got)
		})
	}
}
