package redis

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort            = 6379
	defaultDialTimeout     = 5 * time.Second
	defaultReadTimeout     = 3 * time.Second
	defaultWriteTimeout    = 3 * time.Second
	defaultPoolSize        = 10
	defaultMinIdleConns    = 1
	defaultPoolTimeout     = 4 * time.Second
	defaultConnMaxIdleTime = 30 * time.Minute
	defaultConnMaxLifetime = time.Hour
	defaultMaxRetries      = 3
	defaultMinRetryBackoff = 8 * time.Millisecond
	defaultMaxRetryBackoff = 512 * time.Millisecond
)

type Config struct {
	Connection ConnectionConfig
	TLS        TLSConfig

	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	PoolSize        int
	MinIdleConns    int
	PoolTimeout     time.Duration
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration

	MaxRetries      int
	MinRetryBackoff time.Duration
	MaxRetryBackoff time.Duration
}

type ConnectionConfig struct {
	Host     string
	Port     int
	Database int
	Username string
	Password string
}

type TLSConfig struct {
	Enabled bool

	ServerName string
	CAFile     string
	CertFile   string
	KeyFile    string
}

func DefaultConfig() Config {
	return Config{
		Connection: ConnectionConfig{
			Port: defaultPort,
		},
		DialTimeout:     defaultDialTimeout,
		ReadTimeout:     defaultReadTimeout,
		WriteTimeout:    defaultWriteTimeout,
		PoolSize:        defaultPoolSize,
		MinIdleConns:    defaultMinIdleConns,
		PoolTimeout:     defaultPoolTimeout,
		ConnMaxIdleTime: defaultConnMaxIdleTime,
		ConnMaxLifetime: defaultConnMaxLifetime,
		MaxRetries:      defaultMaxRetries,
		MinRetryBackoff: defaultMinRetryBackoff,
		MaxRetryBackoff: defaultMaxRetryBackoff,
	}
}

func (c Config) Validate() error {
	if err := c.Connection.Validate(); err != nil {
		return err
	}

	if err := c.TLS.Validate(); err != nil {
		return err
	}

	switch {
	case c.DialTimeout <= 0:
		return errors.New("redis dial timeout must be positive")
	case c.ReadTimeout <= 0:
		return errors.New("redis read timeout must be positive")
	case c.WriteTimeout <= 0:
		return errors.New("redis write timeout must be positive")
	case c.PoolSize <= 0:
		return errors.New("redis pool size must be positive")
	case c.MinIdleConns < 0:
		return errors.New("redis min idle conns must not be negative")
	case c.MinIdleConns > c.PoolSize:
		return errors.New("redis min idle conns must not be greater than pool size")
	case c.PoolTimeout <= 0:
		return errors.New("redis pool timeout must be positive")
	case c.ConnMaxIdleTime <= 0:
		return errors.New("redis conn max idle time must be positive")
	case c.ConnMaxLifetime <= 0:
		return errors.New("redis conn max lifetime must be positive")
	case c.MaxRetries < -1:
		return errors.New("redis max retries must not be less than -1")
	case c.MinRetryBackoff < -1:
		return errors.New("redis min retry backoff must not be less than -1")
	case c.MaxRetryBackoff < -1:
		return errors.New("redis max retry backoff must not be less than -1")
	case c.MinRetryBackoff >= 0 &&
		c.MaxRetryBackoff >= 0 &&
		c.MinRetryBackoff > c.MaxRetryBackoff:
		return errors.New("redis min retry backoff must not be greater than max retry backoff")
	default:
		return nil
	}
}

func (c ConnectionConfig) Validate() error {
	switch {
	case strings.TrimSpace(c.Host) == "":
		return errors.New("redis host is required")
	case c.Port <= 0 || c.Port > 65535:
		return errors.New("redis port must be between 1 and 65535")
	case c.Database < 0:
		return errors.New("redis database must not be negative")
	default:
		return nil
	}
}

func (c ConnectionConfig) Address() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

func (c TLSConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	hasCert := strings.TrimSpace(c.CertFile) != ""
	hasKey := strings.TrimSpace(c.KeyFile) != ""

	switch {
	case hasCert != hasKey:
		return errors.New("redis tls cert file and key file must be specified together")
	default:
		return nil
	}
}
