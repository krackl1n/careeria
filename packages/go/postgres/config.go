package postgres

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultMaxConns          int32         = 10
	defaultMinConns          int32         = 1
	defaultMaxConnLifetime   time.Duration = time.Hour
	defaultMaxConnIdleTime   time.Duration = 30 * time.Minute
	defaultHealthCheckPeriod time.Duration = time.Minute
)

type Config struct {
	Connection ConnectionConfig

	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

type ConnectionConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
}

func (c ConnectionConfig) Validate() error {
	switch {
	case strings.TrimSpace(c.Host) == "":
		return errors.New("postgres host is required")
	case c.Port <= 0 || c.Port > 65535:
		return errors.New("postgres port must be between 1 and 65535")
	case strings.TrimSpace(c.Database) == "":
		return errors.New("postgres database is required")
	case strings.TrimSpace(c.User) == "":
		return errors.New("postgres user is required")
	case strings.TrimSpace(c.SSLMode) == "":
		return errors.New("postgres ssl mode is required")
	default:
		return nil
	}
}

func (c ConnectionConfig) DSN() string {
	dsn := &url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		Path:   c.Database,
	}

	if c.Password == "" {
		dsn.User = url.User(c.User)
	} else {
		dsn.User = url.UserPassword(c.User, c.Password)
	}

	query := dsn.Query()
	query.Set("sslmode", c.SSLMode)
	dsn.RawQuery = query.Encode()

	return dsn.String()
}

func (c Config) DSN() string {
	return c.Connection.DSN()
}

func DefaultConfig() Config {
	return Config{
		MaxConns:          defaultMaxConns,
		MinConns:          defaultMinConns,
		MaxConnLifetime:   defaultMaxConnLifetime,
		MaxConnIdleTime:   defaultMaxConnIdleTime,
		HealthCheckPeriod: defaultHealthCheckPeriod,
	}
}

func (c Config) Validate() error {
	if err := c.Connection.Validate(); err != nil {
		return err
	}

	switch {
	case c.MaxConns <= 0:
		return errors.New("postgres max conns must be positive")
	case c.MinConns < 0:
		return errors.New("postgres min conns must not be negative")
	case c.MinConns > c.MaxConns:
		return errors.New("postgres min conns must not be greater than max conns")
	case c.MaxConnLifetime <= 0:
		return errors.New("postgres max conn lifetime must be positive")
	case c.MaxConnIdleTime <= 0:
		return errors.New("postgres max conn idle time must be positive")
	case c.HealthCheckPeriod <= 0:
		return errors.New("postgres health check period must be positive")
	default:
		return nil
	}
}
