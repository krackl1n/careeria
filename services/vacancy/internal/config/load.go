package config

import (
	"fmt"

	sharedconfig "github.com/krackl1n/careeria/config"
)

func Load() (Config, error) {
	var cfg Config
	if err := sharedconfig.LoadEnv("VACANCY", &cfg); err != nil {
		return Config{}, fmt.Errorf("load vacancy config: %w", err)
	}
	// Older .env files may still enable the retired Vacancy health endpoint.
	cfg.GRPCServer.EnableHealth = false
	return cfg, nil
}
