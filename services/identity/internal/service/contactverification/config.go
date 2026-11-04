package contactverification

import (
	"errors"
	"time"
)

type Config struct {
	Lifetime    time.Duration
	MaxAttempts int
}

func (c Config) Validate() error {
	if c.Lifetime < time.Second {
		return errors.New("challenge lifetime must be at least one second")
	}
	if c.Lifetime%time.Second != 0 {
		return errors.New("challenge lifetime must be a whole number of seconds")
	}
	if c.MaxAttempts <= 0 {
		return errors.New("challenge max attempts must be positive")
	}

	return nil
}
