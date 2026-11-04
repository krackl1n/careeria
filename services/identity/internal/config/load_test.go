package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Setenv("IDENTITY_POSTGRES_CONNECTION_HOST", "postgres.internal")
	t.Setenv("IDENTITY_POSTGRES_CONNECTION_PASSWORD", "secret")
	t.Setenv("IDENTITY_ACCESS_TOKEN_LIFETIME", "20m")

	path := writeConfig(t, `
logger:
  level: info
  format: json
  service: identity
grpc_server:
  addr: :50052
shutdown:
  timeout: 15s
redis:
  url: redis://localhost:6379/0
authentication_redirect_uri: http://localhost:3000/login
postgres:
  connection:
    host: localhost
    port: 5432
    database: identity
    user: identity
    ssl_mode: require
  max_conns: 10
  min_conns: 1
  max_conn_lifetime: 1h
  max_conn_idle_time: 30m
  health_check_period: 1m
access_token:
  issuer: https://identity.example.com
  audience: [careeria-api]
  algorithm: EdDSA
  lifetime: 15m
id_token:
  issuer: https://identity.example.com
  algorithm: EdDSA
  lifetime: 15m
contact_verification:
  lifetime: 10m
  max_attempts: 5
jwt_signer:
  key_id: identity-local-1
  private_key_ref: local://access-token-signing-key
  key_file: configs/access-token-signing-key.pem
oauth:
  authorization_lifetime: 10m
  code_lifetime: 2m
  session_idle_lifetime: 24h
  session_lifetime: 720h
  refresh_token_lifetime: 24h
  refresh_family_lifetime: 720h
`)

	cfg, err := load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Postgres.MaxConns != 10 || cfg.Postgres.HealthCheckPeriod != time.Minute {
		t.Fatalf("unexpected postgres config: %+v", cfg.Postgres)
	}
	if cfg.AccessToken.Algorithm != "EdDSA" || cfg.AccessToken.Lifetime != 20*time.Minute {
		t.Fatalf("unexpected access token config: %+v", cfg.AccessToken)
	}
	if cfg.ContactVerification.Lifetime != 10*time.Minute || cfg.ContactVerification.MaxAttempts != 5 {
		t.Fatalf("unexpected contact verification config: %+v", cfg.ContactVerification)
	}
	if len(cfg.AccessToken.Audience) != 1 || cfg.AccessToken.Audience[0] != "careeria-api" {
		t.Fatalf("unexpected access token audience: %#v", cfg.AccessToken.Audience)
	}
	if cfg.JWTSigner.PrivateKeyRef != "local://access-token-signing-key" {
		t.Fatalf("unexpected JWT signer config: %+v", cfg.JWTSigner)
	}
	if cfg.OAuth.SessionLifetime != 30*24*time.Hour {
		t.Fatalf("unexpected OAuth config: %+v", cfg.OAuth)
	}

	const wantDSN = "postgres://identity:secret@postgres.internal:5432/identity?sslmode=require"
	if got := cfg.Postgres.DSN(); got != wantDSN {
		t.Fatalf("unexpected postgres DSN: got %q, want %q", got, wantDSN)
	}
}

func TestLoadRequiresPostgresConfiguration(t *testing.T) {
	path := writeConfig(t, `
logger:
  level: info
  format: json
  service: identity
grpc_server:
  addr: :50052
shutdown:
  timeout: 15s
redis:
  url: redis://localhost:6379/0
authentication_redirect_uri: http://localhost:3000/login
postgres: {}
access_token:
  issuer: https://identity.example.com
  audience: [careeria-api]
  algorithm: EdDSA
  lifetime: 15m
id_token:
  issuer: https://identity.example.com
  algorithm: EdDSA
  lifetime: 15m
contact_verification:
  lifetime: 10m
  max_attempts: 5
jwt_signer:
  key_id: identity-local-1
  private_key_ref: local://access-token-signing-key
  key_file: configs/access-token-signing-key.pem
oauth:
  authorization_lifetime: 10m
  code_lifetime: 2m
  session_idle_lifetime: 24h
  session_lifetime: 720h
  refresh_token_lifetime: 24h
  refresh_family_lifetime: 720h
`)

	if _, err := load(path); err == nil {
		t.Fatal("expected configuration validation error")
	}
}

func TestLoadUsesConfigFileFromEnvironment(t *testing.T) {
	t.Setenv("IDENTITY_POSTGRES_CONNECTION_PASSWORD", "secret")

	path := writeConfig(t, `
logger:
  level: info
  format: json
  service: identity
grpc_server:
  addr: :50052
shutdown:
  timeout: 15s
redis:
  url: redis://localhost:6379/0
authentication_redirect_uri: http://localhost:3000/login
postgres:
  connection:
    host: localhost
    port: 5432
    database: identity
    user: identity
    ssl_mode: disable
  max_conns: 10
  min_conns: 1
  max_conn_lifetime: 1h
  max_conn_idle_time: 30m
  health_check_period: 1m
access_token:
  issuer: https://identity.example.com
  audience: [careeria-api]
  algorithm: EdDSA
  lifetime: 15m
id_token:
  issuer: https://identity.example.com
  algorithm: EdDSA
  lifetime: 15m
contact_verification:
  lifetime: 10m
  max_attempts: 5
jwt_signer:
  key_id: identity-local-1
  private_key_ref: local://access-token-signing-key
  key_file: configs/access-token-signing-key.pem
oauth:
  authorization_lifetime: 10m
  code_lifetime: 2m
  session_idle_lifetime: 24h
  session_lifetime: 720h
  refresh_token_lifetime: 24h
  refresh_family_lifetime: 720h
`)
	t.Setenv(configFileEnv, path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Logger.Service != "identity" {
		t.Fatalf("unexpected service: %q", cfg.Logger.Service)
	}
}

func writeConfig(t *testing.T, value string) string {
	t.Helper()
	value += `
identity_topic: identity.events
outbox:
  batch_size: 100
  poll_interval: 1s
  publish_timeout: 5s
kafka_producer:
  connection:
    brokers: [localhost:19092]
    client_id: identity-test
    version: 3.9.0
  compression: none
  retry_max: 1
  retry_backoff: 10ms
  ack_timeout: 1s
  max_message_bytes: 1048576
`

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return path
}
