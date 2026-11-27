package config

import (
	"testing"
	"time"
)

func TestLoadServiceConfig(t *testing.T) {
	t.Setenv("VACANCY_LOGGER_LEVEL", "info")
	t.Setenv("VACANCY_LOGGER_FORMAT", "json")
	t.Setenv("VACANCY_LOGGER_SERVICE", "vacancy")
	t.Setenv("VACANCY_GRPC_SERVER_ADDR", "127.0.0.1:51054")
	t.Setenv("VACANCY_POSTGRES_CONNECTION_HOST", "vacancy-db")
	t.Setenv("VACANCY_POSTGRES_CONNECTION_PORT", "5432")
	t.Setenv("VACANCY_POSTGRES_CONNECTION_DATABASE", "vacancy")
	t.Setenv("VACANCY_POSTGRES_CONNECTION_USER", "vacancy")
	t.Setenv("VACANCY_POSTGRES_CONNECTION_PASSWORD", "vacancy")
	t.Setenv("VACANCY_POSTGRES_CONNECTION_SSL_MODE", "disable")
	t.Setenv("VACANCY_POSTGRES_MAX_CONNS", "10")
	t.Setenv("VACANCY_POSTGRES_MIN_CONNS", "1")
	t.Setenv("VACANCY_POSTGRES_MAX_CONN_LIFETIME", "1h")
	t.Setenv("VACANCY_POSTGRES_MAX_CONN_IDLE_TIME", "30m")
	t.Setenv("VACANCY_POSTGRES_HEALTH_CHECK_PERIOD", "1m")
	t.Setenv("VACANCY_SHUTDOWN_TIMEOUT", "20s")
	t.Setenv("VACANCY_AUTHORIZATION_CLIENT_TARGET", "openfga:8081")
	t.Setenv("VACANCY_AUTHORIZATION_CLIENT_INSECURE", "true")
	t.Setenv("VACANCY_AUTHORIZATION_STORE_ID", "store")
	t.Setenv("VACANCY_AUTHORIZATION_AUTHORIZATION_MODEL_ID", "model")
	t.Setenv("VACANCY_AUTHORIZATION_TIMEOUT", "5s")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GRPCServer.Addr != "127.0.0.1:51054" || cfg.GRPCServer.EnableHealth {
		t.Fatalf("unexpected gRPC config: %+v", cfg.GRPCServer)
	}
	if cfg.Postgres.Connection.Host != "vacancy-db" || cfg.Postgres.MaxConns != 10 {
		t.Error("unexpected PostgreSQL configuration")
	}
	if cfg.AuthorizationClient.Target != "openfga:8081" || cfg.Authorization.StoreID != "store" || cfg.Authorization.AuthorizationModelID != "model" || cfg.Authorization.Timeout != 5*time.Second {
		t.Fatalf("unexpected Authorization config: %+v %+v", cfg.AuthorizationClient, cfg.Authorization)
	}
	if cfg.Shutdown.Timeout != 20*time.Second {
		t.Errorf("unexpected shutdown timeout: %v", cfg.Shutdown.Timeout)
	}

	t.Setenv("VACANCY_SHUTDOWN_TIMEOUT", "0s")
	if _, err := Load(); err == nil {
		t.Error("expected invalid shutdown timeout to fail validation")
	}
}
