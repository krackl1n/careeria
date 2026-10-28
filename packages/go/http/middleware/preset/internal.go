package preset

import (
	"net/http"

	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/middleware"
)

type InternalConfig struct {
	ErrorHandler handler.ErrorHandler

	BodyLimit int64

	Authenticator middleware.InternalAuthenticator
}

func Internal(config InternalConfig) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middleware.BodyLimit(config.BodyLimit, config.ErrorHandler),
		middleware.InternalAuth(config.Authenticator, config.ErrorHandler),
	}
}
