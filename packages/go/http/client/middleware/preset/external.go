package preset

import (
	"context"
	"net/http"

	clientmw "github.com/krackl1n/careeria/http/client/middleware"
)

type ExternalConfig struct {
	BaseConfig

	APIKeyHeader string
	APIKey       func(context.Context) (string, error)

	TokenProvider clientmw.TokenProvider
}

func External(cfg ExternalConfig) []func(http.RoundTripper) http.RoundTripper {
	middlewares := Base(cfg.BaseConfig)

	if cfg.TokenProvider != nil {
		middlewares = append(middlewares, clientmw.BearerToken(cfg.TokenProvider))
	}

	if cfg.APIKeyHeader != "" && cfg.APIKey != nil {
		middlewares = append(middlewares, clientmw.APIKey(cfg.APIKeyHeader, cfg.APIKey))
	}

	return middlewares
}
