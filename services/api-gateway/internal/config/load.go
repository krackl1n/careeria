package config

import (
	"fmt"
	"os"
	"strings"

	sharedconfig "github.com/krackl1n/careeria/config"
)

const (
	envPrefix     = "API_GATEWAY"
	configFileEnv = "API_GATEWAY_CONFIG_FILE"
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
		File:      file,
		EnvPrefix: envPrefix,
	}, &cfg); err != nil {
		return Config{}, fmt.Errorf("load API gateway config: %w", err)
	}

	return cfg, nil
}
