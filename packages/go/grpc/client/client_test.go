package client

import (
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
			name: "missing target",
			cfg: Config{
				Insecure: true,
			},
			wantErr: "grpc target is required",
		},
		{
			name: "insecure and tls together",
			cfg: Config{
				Target:   "localhost:8080",
				Insecure: true,
				TLS: TLSConfig{
					Enabled: true,
				},
			},
			wantErr: "grpc insecure and tls cannot be enabled together",
		},
		{
			name: "secure without tls",
			cfg: Config{
				Target: "localhost:8080",
			},
			wantErr: "grpc tls must be enabled when insecure is false",
		},
		{
			name: "negative recv size",
			cfg: Config{
				Target:             "localhost:8080",
				Insecure:           true,
				MaxRecvMessageSize: -1,
			},
			wantErr: "grpc max recv message size must not be negative",
		},
		{
			name: "missing client key",
			cfg: Config{
				Target: "localhost:8080",
				TLS: TLSConfig{
					Enabled:        true,
					ClientCertFile: "cert.pem",
				},
			},
			wantErr: "grpc client key file is required",
		},
		{
			name: "valid insecure",
			cfg: Config{
				Target:   "localhost:8080",
				Insecure: true,
				Keepalive: KeepaliveConfig{
					Time:    time.Second,
					Timeout: 2 * time.Second,
				},
			},
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

func TestBuildDialOptions(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Target:             "localhost:8080",
		Insecure:           true,
		UserAgent:          "svc",
		MaxRecvMessageSize: 1024,
		MaxSendMessageSize: 2048,
		Keepalive: KeepaliveConfig{
			Enabled:             true,
			Time:                5 * time.Second,
			Timeout:             7 * time.Second,
			PermitWithoutStream: true,
		},
		Tracing: TracingConfig{
			Enabled: true,
		},
	}

	options, err := buildDialOptions(cfg)
	require.NoError(t, err)
	require.Len(t, options, 5)
}

func TestBuildTransportOption(t *testing.T) {
	t.Parallel()

	t.Run("insecure", func(t *testing.T) {
		t.Parallel()
		option, err := buildTransportOption(Config{Insecure: true})
		require.NoError(t, err)
		require.NotNil(t, option)
	})

	t.Run("tls error", func(t *testing.T) {
		t.Parallel()
		option, err := buildTransportOption(Config{
			TLS: TLSConfig{
				Enabled: true,
				CAFile:  "/missing/ca.pem",
			},
		})
		require.Nil(t, option)
		require.Error(t, err)
		require.Contains(t, err.Error(), "create transport credentials")
	})
}

func TestHelpers(t *testing.T) {
	t.Parallel()

	require.Equal(t, 3*time.Second, durationOrDefault(3*time.Second, time.Second))
	require.Equal(t, time.Second, durationOrDefault(0, time.Second))

	require.Len(t, buildCallOptions(Config{}), 0)
	require.Len(t, buildCallOptions(Config{
		MaxRecvMessageSize: 10,
		MaxSendMessageSize: 20,
	}), 2)

	require.NotNil(t, buildKeepaliveOption(KeepaliveConfig{}))
}

func TestNew(t *testing.T) {
	t.Parallel()

	conn, err := New(Config{})
	require.Nil(t, conn)
	require.EqualError(t, err, "validate grpc config: grpc target is required")
}
