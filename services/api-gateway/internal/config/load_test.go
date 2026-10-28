package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Setenv("API_GATEWAY_HTTP_ADDR", ":8081")
	t.Setenv("API_GATEWAY_SWAGGER_TITLE", "Careeria API (local)")
	t.Setenv("API_GATEWAY_IDENTITY_GRPC_TARGET", "identity:9096")

	path := writeConfig(t, `
http:
  addr: ":8080"
  read_timeout: 5s
  read_header_timeout: 1s
  write_timeout: 10s
  idle_timeout: 30s
logger:
  level: info
  format: json
  service: api-gateway
shutdown:
  timeout: 15s
metrics:
  enabled: true
  addr: ":9090"
  namespace: careeria
  read_header_timeout: 5s
tracing:
  enabled: true
  service_name: api-gateway
  exporter: none
  sampling_ratio: 1
  export_timeout: 5s
  shutdown_timeout: 5s
  batch_timeout: 5s
  max_queue_size: 2048
  max_export_batch_size: 512
gateway:
  base:
    trace_operation: api-gateway.http
  body_limit: 1048576
  compression_level: 5
swagger:
  enabled: true
  title: Careeria API
  path: /swagger/
  specification_path: /openapi/careeria.swagger.json
identity_grpc:
  target: "identity:9095"
  insecure: true
authentication:
  issuer: "https://identity.example"
  audience: ["careeria-api"]
  jwks_cache:
    ttl: 5m
    refresh_timeout: 2s
    refresh_cooldown: 10s
`)

	cfg, err := load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.HTTP.Addr != ":8081" {
		t.Fatalf("unexpected HTTP address: %q", cfg.HTTP.Addr)
	}

	if cfg.HTTP.ReadTimeout != 5*time.Second {
		t.Fatalf("unexpected read timeout: %s", cfg.HTTP.ReadTimeout)
	}

	if cfg.HTTP.ReadHeaderTimeout != time.Second {
		t.Fatalf("unexpected read header timeout: %s", cfg.HTTP.ReadHeaderTimeout)
	}

	if cfg.HTTP.WriteTimeout != 10*time.Second {
		t.Fatalf("unexpected write timeout: %s", cfg.HTTP.WriteTimeout)
	}

	if cfg.HTTP.IdleTimeout != 30*time.Second {
		t.Fatalf("unexpected idle timeout: %s", cfg.HTTP.IdleTimeout)
	}

	if cfg.Shutdown.Timeout != 15*time.Second {
		t.Fatalf("unexpected shutdown timeout: %s", cfg.Shutdown.Timeout)
	}

	if !cfg.Metrics.Enabled || cfg.Metrics.Addr != ":9090" {
		t.Fatalf("unexpected metrics config: %+v", cfg.Metrics)
	}

	if !cfg.Tracing.Enabled || cfg.Tracing.ServiceName != "api-gateway" {
		t.Fatalf("unexpected tracing config: %+v", cfg.Tracing)
	}

	if cfg.IdentityGRPC.Target != "identity:9096" {
		t.Fatalf("unexpected identity gRPC target: %q", cfg.IdentityGRPC.Target)
	}
	if !cfg.IdentityGRPC.Insecure {
		t.Fatal("expected insecure identity gRPC client")
	}
	if cfg.Authentication.JWKSCache.TTL != 5*time.Minute ||
		cfg.Authentication.JWKSCache.RefreshTimeout != 2*time.Second ||
		cfg.Authentication.JWKSCache.RefreshCooldown != 10*time.Second {
		t.Fatalf("unexpected JWKS cache config: %+v", cfg.Authentication.JWKSCache)
	}

	if !cfg.Swagger.Enabled || cfg.Swagger.Title != "Careeria API (local)" {
		t.Fatalf("unexpected Swagger config: %+v", cfg.Swagger)
	}
}

func TestLoadRequiresHTTPAddress(t *testing.T) {
	path := writeConfig(t, "http: {}\n")

	if _, err := load(path); err == nil {
		t.Fatal("expected configuration validation error")
	}
}

func TestLoadUsesConfigFileFromEnvironment(t *testing.T) {
	path := writeConfig(t, `
http:
  addr: ":8080"
logger:
  level: info
  format: json
  service: api-gateway
shutdown:
  timeout: 15s
gateway:
  base:
    trace_operation: api-gateway.http
  body_limit: 1048576
  compression_level: 5
identity_grpc:
  target: "identity:9095"
  insecure: true
authentication:
  issuer: "https://identity.example"
  audience: ["careeria-api"]
  jwks_cache:
    ttl: 5m
    refresh_timeout: 2s
    refresh_cooldown: 10s
`)
	t.Setenv(configFileEnv, path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("unexpected HTTP address: %q", cfg.HTTP.Addr)
	}
}

func writeConfig(t *testing.T, value string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return path
}
