package preset

import (
	"net/http"

	clientmw "github.com/krackl1n/careeria/http/client/middleware"
)

type InternalConfig struct {
	BaseConfig

	TokenProvider clientmw.TokenProvider
}

func Internal(cfg InternalConfig) []func(http.RoundTripper) http.RoundTripper {
	middlewares := Base(cfg.BaseConfig)

	if cfg.TokenProvider != nil {
		middlewares = append(middlewares, clientmw.BearerToken(cfg.TokenProvider))
	}

	return middlewares
}
