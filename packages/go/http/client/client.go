package client

import "net/http"

type TransportMiddleware func(http.RoundTripper) http.RoundTripper

func New(cfg Config, middlewares ...TransportMiddleware) *http.Client {
	transport := buildTransport(cfg)

	var roundTripper http.RoundTripper = transport

	for i := len(middlewares) - 1; i >= 0; i-- {
		roundTripper = middlewares[i](roundTripper)
	}

	return &http.Client{
		Timeout:   cfg.Timeout,
		Transport: roundTripper,
	}
}
