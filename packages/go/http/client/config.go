package client

import "time"

type Config struct {
	Timeout time.Duration

	DialTimeout   time.Duration
	DialKeepAlive time.Duration

	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration

	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	ExpectContinueTimeout time.Duration

	DisableCompression bool
}
