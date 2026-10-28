package server

import (
	"errors"
	"time"
)

type Config struct {
	Addr string

	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func (c Config) Validate() error {
	if c.Addr == "" {
		return errors.New("http server addr is required")
	}

	return nil
}
