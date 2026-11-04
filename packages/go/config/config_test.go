package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fixtureConfig struct {
	HTTP     fixtureHTTPConfig
	Postgres fixturePostgresConfig
	Kafka    fixtureKafkaConfig
}

type fixtureHTTPConfig struct {
	Address         string
	ShutdownTimeout time.Duration
}

type fixturePostgresConfig struct {
	Host     string
	Password string
}

type fixtureKafkaConfig struct {
	Brokers []string
}

func (c *fixtureConfig) Validate() error {
	switch {
	case c.HTTP.Address == "":
		return errors.New("http address is required")
	case c.HTTP.ShutdownTimeout <= 0:
		return errors.New("http shutdown timeout must be positive")
	case c.Postgres.Host == "":
		return errors.New("postgres host is required")
	case c.Postgres.Password == "":
		return errors.New("postgres password is required")
	case len(c.Kafka.Brokers) == 0:
		return errors.New("kafka brokers are required")
	default:
		return nil
	}
}

func TestLoad(t *testing.T) {
	path := writeConfig(t, `
http:
  address: ":8080"
  shutdown_timeout: 5s

postgres:
  host: postgres

kafka:
  brokers:
    - kafka-1:9092
    - kafka-2:9092
`)

	t.Setenv("APP_HTTP_ADDRESS", ":8081")
	t.Setenv("APP_POSTGRES_PASSWORD", "secret")

	var cfg fixtureConfig

	err := Load(Options{
		File:      path,
		EnvPrefix: "APP",
		EnvOnly: []string{
			"postgres.password",
		},
		RequiredEnv: []string{
			"postgres.password",
		},
	}, &cfg)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.HTTP.Address != ":8081" {
		t.Fatalf("unexpected http address: %q", cfg.HTTP.Address)
	}

	if cfg.HTTP.ShutdownTimeout != 5*time.Second {
		t.Fatalf(
			"unexpected http shutdown timeout: %s",
			cfg.HTTP.ShutdownTimeout,
		)
	}

	if cfg.Postgres.Password != "secret" {
		t.Fatalf("unexpected postgres password: %q", cfg.Postgres.Password)
	}

	if len(cfg.Kafka.Brokers) != 2 {
		t.Fatalf("unexpected kafka brokers: %#v", cfg.Kafka.Brokers)
	}
}

func TestLoadOverridesStringSliceFromEnvironment(t *testing.T) {
	path := writeConfig(t, `
http:
  address: ":8080"
  shutdown_timeout: 5s

postgres:
  host: postgres

kafka:
  brokers:
    - kafka-1:9092
`)

	t.Setenv("APP_POSTGRES_PASSWORD", "secret")
	t.Setenv("APP_KAFKA_BROKERS", "kafka-2:9092,kafka-3:9092")

	var cfg fixtureConfig

	err := Load(Options{
		File:      path,
		EnvPrefix: "APP",
		EnvOnly: []string{
			"postgres.password",
		},
		RequiredEnv: []string{
			"postgres.password",
		},
	}, &cfg)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if len(cfg.Kafka.Brokers) != 2 {
		t.Fatalf("unexpected kafka brokers: %#v", cfg.Kafka.Brokers)
	}
	if cfg.Kafka.Brokers[0] != "kafka-2:9092" || cfg.Kafka.Brokers[1] != "kafka-3:9092" {
		t.Fatalf("unexpected kafka brokers: %#v", cfg.Kafka.Brokers)
	}
}

func TestLoadRejectsUnknownYAMLKey(t *testing.T) {
	path := writeConfig(t, `
http:
  address: ":8080"
  shutdown_timout: 5s

postgres:
  host: postgres

kafka:
  brokers:
    - kafka-1:9092
`)

	t.Setenv("APP_POSTGRES_PASSWORD", "secret")

	var cfg fixtureConfig

	err := Load(Options{
		File:      path,
		EnvPrefix: "APP",
		EnvOnly: []string{
			"postgres.password",
		},
		RequiredEnv: []string{
			"postgres.password",
		},
	}, &cfg)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadRejectsEnvOnlyValueInYAML(t *testing.T) {
	path := writeConfig(t, `
http:
  address: ":8080"
  shutdown_timeout: 5s

postgres:
  host: postgres
  password: yaml-secret

kafka:
  brokers:
    - kafka-1:9092
`)

	t.Setenv("APP_POSTGRES_PASSWORD", "secret")

	var cfg fixtureConfig

	err := Load(Options{
		File:      path,
		EnvPrefix: "APP",
		EnvOnly: []string{
			"postgres.password",
		},
		RequiredEnv: []string{
			"postgres.password",
		},
	}, &cfg)

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "must not be present in YAML") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRequiresEnvironmentVariable(t *testing.T) {
	path := writeConfig(t, `
http:
  address: ":8080"
  shutdown_timeout: 5s

postgres:
  host: postgres

kafka:
  brokers:
    - kafka-1:9092
`)

	var cfg fixtureConfig

	err := Load(Options{
		File:      path,
		EnvPrefix: "APP",
		EnvOnly: []string{
			"postgres.password",
		},
		RequiredEnv: []string{
			"postgres.password",
		},
	}, &cfg)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSnakeCase(t *testing.T) {
	cases := map[string]string{
		"Address":         "address",
		"HTTPAddress":     "http_address",
		"GRPCPort":        "grpc_port",
		"OTLPEndpoint":    "otlp_endpoint",
		"SASLPassword":    "sasl_password",
		"OAuthClientID":   "oauth_client_id",
		"JWKSCache":       "jwks_cache",
		"CacheTTL":        "cache_ttl",
		"HTTP2Enabled":    "http2_enabled",
		"TLSConfig":       "tls_config",
		"ShutdownTimeout": "shutdown_timeout",
	}

	for input, expected := range cases {
		if actual := snakeCase(input); actual != expected {
			t.Fatalf("%s: expected %q, got %q", input, expected, actual)
		}
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
