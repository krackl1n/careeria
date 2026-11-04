package migration

import (
	"fmt"
	"io/fs"
	"time"
)

type Config struct {
	DSN         string
	ServiceName string
	Timeout     time.Duration
	FS          fs.FS
}

func (c Config) Validate() error {
	switch {
	case c.DSN == "":
		return fmt.Errorf("dsn is required")

	case c.ServiceName == "":
		return fmt.Errorf("service name is required")

	case c.Timeout <= 0:
		return fmt.Errorf("timeout must be greater than zero")

	case c.FS == nil:
		return fmt.Errorf("migrations filesystem is required")

	default:
		return nil
	}
}
