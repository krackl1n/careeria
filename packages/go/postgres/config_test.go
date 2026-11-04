package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	require.Equal(t, defaultMaxConns, cfg.MaxConns)
	require.Equal(t, defaultMinConns, cfg.MinConns)
	require.Equal(t, defaultMaxConnLifetime, cfg.MaxConnLifetime)
	require.Equal(t, defaultMaxConnIdleTime, cfg.MaxConnIdleTime)
	require.Equal(t, defaultHealthCheckPeriod, cfg.HealthCheckPeriod)
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	valid := DefaultConfig()
	valid.Connection = ConnectionConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "app",
		User:     "user",
		Password: "password",
		SSLMode:  "disable",
	}

	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "missing postgres host",
			cfg:     DefaultConfig(),
			wantErr: "postgres host is required",
		},
		{
			name: "max conns must be positive",
			cfg: func() Config {
				cfg := valid
				cfg.MaxConns = 0
				return cfg
			}(),
			wantErr: "postgres max conns must be positive",
		},
		{
			name: "min conns must not be negative",
			cfg: func() Config {
				cfg := valid
				cfg.MinConns = -1
				return cfg
			}(),
			wantErr: "postgres min conns must not be negative",
		},
		{
			name: "min conns must not exceed max conns",
			cfg: func() Config {
				cfg := valid
				cfg.MinConns = cfg.MaxConns + 1
				return cfg
			}(),
			wantErr: "postgres min conns must not be greater than max conns",
		},
		{
			name: "max conn lifetime must be positive",
			cfg: func() Config {
				cfg := valid
				cfg.MaxConnLifetime = 0
				return cfg
			}(),
			wantErr: "postgres max conn lifetime must be positive",
		},
		{
			name: "max conn idle time must be positive",
			cfg: func() Config {
				cfg := valid
				cfg.MaxConnIdleTime = 0
				return cfg
			}(),
			wantErr: "postgres max conn idle time must be positive",
		},
		{
			name: "health check period must be positive",
			cfg: func() Config {
				cfg := valid
				cfg.HealthCheckPeriod = 0
				return cfg
			}(),
			wantErr: "postgres health check period must be positive",
		},
		{
			name: "valid",
			cfg: func() Config {
				cfg := valid
				cfg.MaxConnLifetime = time.Minute
				return cfg
			}(),
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestConnectionConfigDSN(t *testing.T) {
	t.Parallel()

	config := ConnectionConfig{
		Host:     "::1",
		Port:     5432,
		Database: "identity service",
		User:     "identity@example.com",
		Password: "secret/password",
		SSLMode:  "disable",
	}

	require.NoError(t, config.Validate())
	require.Equal(
		t,
		"postgres://identity%40example.com:secret%2Fpassword@[::1]:5432/identity%20service?sslmode=disable",
		config.DSN(),
	)
}
