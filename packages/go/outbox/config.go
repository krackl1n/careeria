package outbox

import (
	"errors"
	"time"
)

type Config struct {
	BatchSize      int
	PollInterval   time.Duration
	PublishTimeout time.Duration
}

func (c Config) Validate() error {
	switch {
	case c.BatchSize <= 0:
		return errors.New("batch size must be greater than zero")
	case c.PollInterval <= 0:
		return errors.New("poll interval must be greater than zero")
	case c.PublishTimeout <= 0:
		return errors.New("publish timeout must be greater than zero")
	default:
		return nil
	}
}
