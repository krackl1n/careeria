package client

import (
	"net"
	"net/http"
	"time"
)

func buildTransport(cfg Config) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,

		DialContext: (&net.Dialer{
			Timeout:   durationOrDefault(cfg.DialTimeout, 30*time.Second),
			KeepAlive: durationOrDefault(cfg.DialKeepAlive, 30*time.Second),
		}).DialContext,

		MaxIdleConns:        intOrDefault(cfg.MaxIdleConns, 100),
		MaxIdleConnsPerHost: intOrDefault(cfg.MaxIdleConnsPerHost, 20),
		IdleConnTimeout:     durationOrDefault(cfg.IdleConnTimeout, 90*time.Second),

		TLSHandshakeTimeout:   durationOrDefault(cfg.TLSHandshakeTimeout, 10*time.Second),
		ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
		ExpectContinueTimeout: durationOrDefault(cfg.ExpectContinueTimeout, time.Second),

		DisableCompression: cfg.DisableCompression,
	}
}

func durationOrDefault(value time.Duration, fallback time.Duration) time.Duration {
	if value > 0 {
		return value
	}

	return fallback
}

func intOrDefault(value int, fallback int) int {
	if value > 0 {
		return value
	}

	return fallback
}
