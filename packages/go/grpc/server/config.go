package grpcserver

import (
	"errors"
	"time"
)

type Config struct {
	Addr string

	EnableHealth     bool
	EnableReflection bool

	ConnectionTimeout time.Duration

	MaxRecvMsgSize int
	MaxSendMsgSize int
}

func (c Config) Validate() error {
	if c.Addr == "" {
		return errors.New("grpc server addr is required")
	}

	return nil
}
