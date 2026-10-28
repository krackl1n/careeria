package config

import (
	"fmt"
	"os"
	"strings"

	sharedconfig "github.com/krackl1n/careeria/config"
)

const (
	envPrefix     = "IDENTITY"
	configFileEnv = "IDENTITY_CONFIG_FILE"
	defaultFile   = "configs/config.yaml"
)

func Load() (Config, error) {
	file := strings.TrimSpace(os.Getenv(configFileEnv))
	if file == "" {
		file = defaultFile
	}

	return load(file)
}

func load(file string) (Config, error) {
	var cfg Config

	if err := sharedconfig.Load(sharedconfig.Options{
		File:        file,
		EnvPrefix:   envPrefix,
		EnvOnly:     []string{"postgres.connection.password", "kafka_producer.connection.sasl.password"},
		RequiredEnv: []string{"postgres.connection.password"},
	}, &cfg); err != nil {
		return Config{}, fmt.Errorf("load identity config: %w", err)
	}

	return cfg, nil
}
