package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "missing addr",
			cfg:     Config{},
			wantErr: "http server addr is required",
		},
		{
			name: "valid",
			cfg:  Config{Addr: ":8080"},
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

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("requires handler", func(t *testing.T) {
		t.Parallel()

		server, err := New(Config{Addr: ":8080"}, nil)
		require.Nil(t, server)
		require.EqualError(t, err, "http handler is required")
	})

	t.Run("builds http server", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			Addr:              ":8081",
			ReadTimeout:       time.Second,
			ReadHeaderTimeout: 2 * time.Second,
			WriteTimeout:      3 * time.Second,
			IdleTimeout:       4 * time.Second,
		}

		server, err := New(cfg, http.NewServeMux())
		require.NoError(t, err)
		require.NotNil(t, server)
		require.Equal(t, cfg.Addr, server.HTTP().Addr)
		require.Equal(t, cfg.ReadTimeout, server.HTTP().ReadTimeout)
		require.Equal(t, cfg.ReadHeaderTimeout, server.HTTP().ReadHeaderTimeout)
		require.Equal(t, cfg.WriteTimeout, server.HTTP().WriteTimeout)
		require.Equal(t, cfg.IdleTimeout, server.HTTP().IdleTimeout)
	})
}

func TestServerName(t *testing.T) {
	server, err := New(Config{Addr: ":8080"}, http.NewServeMux())
	require.NoError(t, err)

	require.Equal(t, "http", server.Name())
}

func TestStart(t *testing.T) {
	t.Parallel()

	t.Run("wraps listen error", func(t *testing.T) {
		t.Parallel()

		server, err := New(Config{Addr: "bad-address"}, http.NewServeMux())
		require.NoError(t, err)

		err = server.Start()
		require.Error(t, err)
		require.Contains(t, err.Error(), "serve http:")
	})

	t.Run("returns nil when server is already closed", func(t *testing.T) {
		t.Parallel()

		server, err := New(Config{Addr: "127.0.0.1:0"}, http.NewServeMux())
		require.NoError(t, err)

		require.NoError(t, server.HTTP().Close())
		require.NoError(t, server.Start())
	})
}

func TestShutdown(t *testing.T) {
	t.Parallel()

	server, err := New(Config{Addr: "127.0.0.1:0"}, http.NewServeMux())
	require.NoError(t, err)

	require.NoError(t, server.Shutdown(context.Background()))
}
