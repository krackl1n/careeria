package metrics

import (
	"errors"
	"strings"
	"time"
)

type Config struct {
	Enabled           bool
	Addr              string
	Namespace         string
	ReadHeaderTimeout time.Duration
}

func DefaultConfig(namespace string, addr string) Config {
	return Config{
		Enabled:           true,
		Addr:              addr,
		Namespace:         namespace,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if strings.TrimSpace(c.Addr) == "" {
		return errors.New("addr is required")
	}

	if strings.TrimSpace(c.Namespace) == "" {
		return errors.New("namespace is required")
	}

	if c.ReadHeaderTimeout <= 0 {
		return errors.New("read header timeout must be positive")
	}

	return nil
}
