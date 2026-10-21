package shutdown

import (
	"fmt"
	"time"
)

const DefaultTimeout = 15 * time.Second

type Config struct {
	Timeout time.Duration
}

func DefaultConfig() Config {
	return Config{
		Timeout: DefaultTimeout,
	}
}

func (c Config) Validate() error {
	if c.Timeout <= 0 {
		return fmt.Errorf(
			"%w: got %s",
			ErrInvalidTimeout,
			c.Timeout,
		)
	}

	return nil
}
