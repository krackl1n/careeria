package migration

import (
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	validFS := fstest.MapFS{
		"migrations/00001_init.sql": {
			Data: []byte("-- +goose Up\nselect 1;"),
		},
	}

	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name: "valid",
			config: Config{
				DSN:         "postgres://postgres:postgres@localhost:5432/app",
				ServiceName: "identity",
				Timeout:     time.Minute,
				FS:          validFS,
			},
		},
		{
			name: "dsn required",
			config: Config{
				ServiceName: "identity",
				Timeout:     time.Minute,
				FS:          validFS,
			},
			wantErr: "dsn is required",
		},
		{
			name: "service name required",
			config: Config{
				DSN:     "postgres://postgres:postgres@localhost:5432/app",
				Timeout: time.Minute,
				FS:      validFS,
			},
			wantErr: "service name is required",
		},
		{
			name: "zero timeout",
			config: Config{
				DSN:         "postgres://postgres:postgres@localhost:5432/app",
				ServiceName: "identity",
				FS:          validFS,
			},
			wantErr: "timeout must be greater than zero",
		},
		{
			name: "negative timeout",
			config: Config{
				DSN:         "postgres://postgres:postgres@localhost:5432/app",
				ServiceName: "identity",
				Timeout:     -time.Second,
				FS:          validFS,
			},
			wantErr: "timeout must be greater than zero",
		},
		{
			name: "filesystem required",
			config: Config{
				DSN:         "postgres://postgres:postgres@localhost:5432/app",
				ServiceName: "identity",
				Timeout:     time.Minute,
			},
			wantErr: "migrations filesystem is required",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
		})
	}
}
