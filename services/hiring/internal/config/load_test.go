package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Setenv("HIRING_POSTGRES_CONNECTION_HOST", "postgres.internal")
	t.Setenv("HIRING_POSTGRES_CONNECTION_PASSWORD", "secret")

	path := writeConfig(t, `
logger:
  level: info
  format: json
  service: hiring
grpc_server:
  addr: :50053
metrics:
  enabled: true
  addr: :9091
  namespace: careeria_hiring
  read_header_timeout: 5s
shutdown:
  timeout: 15s
postgres:
  connection:
    host: localhost
    port: 5432
    database: hiring
    user: hiring
    ssl_mode: disable
  max_conns: 10
  min_conns: 1
  max_conn_lifetime: 1h
  max_conn_idle_time: 30m
  health_check_period: 1m
hiring_topic: hiring.events
outbox:
  batch_size: 100
  poll_interval: 1s
  publish_timeout: 5s
`)

	cfg, err := load(path)
	require.NoError(t, err)

	require.Equal(t, "hiring", cfg.Logger.Service)
	require.Equal(t, ":50053", cfg.GRPCServer.Addr)
	require.Equal(t, "postgres.internal", cfg.Postgres.Connection.Host)
	require.Equal(t, "secret", cfg.Postgres.Connection.Password)
	require.Equal(t, "hiring.events", cfg.HiringTopic)
	require.Equal(t, 15*time.Second, cfg.Shutdown.Timeout)
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	err := os.WriteFile(path, []byte(content), 0o600)
	require.NoError(t, err)

	return path
}

