package grpc

import (
	"errors"
	"strings"
	"time"
)

type Config struct {
	StoreID              string
	AuthorizationModelID string
	Timeout              time.Duration
}

func (c Config) Validate() error {
	switch {
	case strings.TrimSpace(c.StoreID) == "":
		return errors.New("openfga store id is required")
	case strings.TrimSpace(c.AuthorizationModelID) == "":
		return errors.New("openfga authorization model id is required")
	case c.Timeout <= 0:
		return errors.New("openfga timeout must be positive")
	default:
		return nil
	}
}
