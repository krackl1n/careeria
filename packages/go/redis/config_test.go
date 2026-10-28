package redis

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	require.Equal(t, Config{
		Connection: ConnectionConfig{
			Port: defaultPort,
		},
		DialTimeout:     defaultDialTimeout,
		ReadTimeout:     defaultReadTimeout,
		WriteTimeout:    defaultWriteTimeout,
		PoolSize:        defaultPoolSize,
		MinIdleConns:    defaultMinIdleConns,
		PoolTimeout:     defaultPoolTimeout,
		ConnMaxIdleTime: defaultConnMaxIdleTime,
		ConnMaxLifetime: defaultConnMaxLifetime,
		MaxRetries:      defaultMaxRetries,
		MinRetryBackoff: defaultMinRetryBackoff,
		MaxRetryBackoff: defaultMaxRetryBackoff,
	}, DefaultConfig())
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	valid := validConfig()

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{name: "valid"},
		{
			name: "connection is invalid",
			mutate: func(cfg *Config) {
				cfg.Connection.Host = " "
			},
			wantErr: "redis host is required",
		},
		{
			name: "tls config is invalid",
			mutate: func(cfg *Config) {
				cfg.TLS = TLSConfig{Enabled: true, CertFile: "client.crt"}
			},
			wantErr: "redis tls cert file and key file must be specified together",
		},
		{
			name: "dial timeout must be positive",
			mutate: func(cfg *Config) {
				cfg.DialTimeout = 0
			},
			wantErr: "redis dial timeout must be positive",
		},
		{
			name: "read timeout must be positive",
			mutate: func(cfg *Config) {
				cfg.ReadTimeout = -time.Second
			},
			wantErr: "redis read timeout must be positive",
		},
		{
			name: "write timeout must be positive",
			mutate: func(cfg *Config) {
				cfg.WriteTimeout = 0
			},
			wantErr: "redis write timeout must be positive",
		},
		{
			name: "pool size must be positive",
			mutate: func(cfg *Config) {
				cfg.PoolSize = 0
			},
			wantErr: "redis pool size must be positive",
		},
		{
			name: "min idle conns must not be negative",
			mutate: func(cfg *Config) {
				cfg.MinIdleConns = -1
			},
			wantErr: "redis min idle conns must not be negative",
		},
		{
			name: "min idle conns must not exceed pool size",
			mutate: func(cfg *Config) {
				cfg.MinIdleConns = cfg.PoolSize + 1
			},
			wantErr: "redis min idle conns must not be greater than pool size",
		},
		{
			name: "pool timeout must be positive",
			mutate: func(cfg *Config) {
				cfg.PoolTimeout = 0
			},
			wantErr: "redis pool timeout must be positive",
		},
		{
			name: "conn max idle time must be positive",
			mutate: func(cfg *Config) {
				cfg.ConnMaxIdleTime = 0
			},
			wantErr: "redis conn max idle time must be positive",
		},
		{
			name: "conn max lifetime must be positive",
			mutate: func(cfg *Config) {
				cfg.ConnMaxLifetime = 0
			},
			wantErr: "redis conn max lifetime must be positive",
		},
		{
			name: "max retries must not be less than minus one",
			mutate: func(cfg *Config) {
				cfg.MaxRetries = -2
			},
			wantErr: "redis max retries must not be less than -1",
		},
		{
			name: "min retry backoff must not be less than minus one",
			mutate: func(cfg *Config) {
				cfg.MinRetryBackoff = -2
			},
			wantErr: "redis min retry backoff must not be less than -1",
		},
		{
			name: "max retry backoff must not be less than minus one",
			mutate: func(cfg *Config) {
				cfg.MaxRetryBackoff = -2
			},
			wantErr: "redis max retry backoff must not be less than -1",
		},
		{
			name: "min retry backoff must not exceed max retry backoff",
			mutate: func(cfg *Config) {
				cfg.MinRetryBackoff = time.Second
				cfg.MaxRetryBackoff = time.Millisecond
			},
			wantErr: "redis min retry backoff must not be greater than max retry backoff",
		},
		{
			name: "minus one retry backoffs disable comparison",
			mutate: func(cfg *Config) {
				cfg.MinRetryBackoff = time.Second
				cfg.MaxRetryBackoff = -1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := valid
			if tt.mutate != nil {
				tt.mutate(&cfg)
			}

			err := cfg.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestConnectionConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     ConnectionConfig
		wantErr string
	}{
		{
			name:    "missing host",
			cfg:     ConnectionConfig{Host: "\t", Port: 6379},
			wantErr: "redis host is required",
		},
		{
			name:    "zero port",
			cfg:     ConnectionConfig{Host: "localhost", Port: 0},
			wantErr: "redis port must be between 1 and 65535",
		},
		{
			name:    "port is too large",
			cfg:     ConnectionConfig{Host: "localhost", Port: 65536},
			wantErr: "redis port must be between 1 and 65535",
		},
		{
			name:    "negative database",
			cfg:     ConnectionConfig{Host: "localhost", Port: 6379, Database: -1},
			wantErr: "redis database must not be negative",
		},
		{
			name: "valid boundary values",
			cfg:  ConnectionConfig{Host: "localhost", Port: 65535},
		},
	}

	for _, tt := range tests {
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

func TestConnectionConfigAddress(t *testing.T) {
	t.Parallel()

	require.Equal(t, "redis.internal:6379", (ConnectionConfig{
		Host: "redis.internal",
		Port: 6379,
	}).Address())
	require.Equal(t, "[2001:db8::1]:6380", (ConnectionConfig{
		Host: "2001:db8::1",
		Port: 6380,
	}).Address())
}

func TestTLSConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     TLSConfig
		wantErr string
	}{
		{
			name: "disabled ignores incomplete client certificate",
			cfg:  TLSConfig{CertFile: "client.crt"},
		},
		{
			name: "enabled without client certificate",
			cfg:  TLSConfig{Enabled: true},
		},
		{
			name: "enabled with certificate pair",
			cfg: TLSConfig{
				Enabled:  true,
				CertFile: "client.crt",
				KeyFile:  "client.key",
			},
		},
		{
			name:    "certificate without key",
			cfg:     TLSConfig{Enabled: true, CertFile: "client.crt"},
			wantErr: "redis tls cert file and key file must be specified together",
		},
		{
			name:    "key without certificate",
			cfg:     TLSConfig{Enabled: true, KeyFile: "client.key"},
			wantErr: "redis tls cert file and key file must be specified together",
		},
		{
			name:    "whitespace certificate is empty",
			cfg:     TLSConfig{Enabled: true, CertFile: " ", KeyFile: "client.key"},
			wantErr: "redis tls cert file and key file must be specified together",
		},
	}

	for _, tt := range tests {
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

func validConfig() Config {
	cfg := DefaultConfig()
	cfg.Connection.Host = "127.0.0.1"
	return cfg
}
