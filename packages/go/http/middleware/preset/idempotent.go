package preset

import (
	"net/http"

	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/middleware"
)

type IdempotentConfig struct {
	ErrorHandler handler.ErrorHandler

	Idempotency middleware.IdempotencyConfig
}

func Idempotent(config IdempotentConfig) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middleware.Idempotency(config.Idempotency, config.ErrorHandler),
	}
}
