package preset

import (
	"net/http"

	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/middleware"
)

type ProtectedConfig struct {
	ErrorHandler handler.ErrorHandler

	BodyLimit int64

	Authenticator middleware.Authenticator
}

func Protected(config ProtectedConfig) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middleware.BodyLimit(config.BodyLimit, config.ErrorHandler),
		middleware.Auth(config.Authenticator, config.ErrorHandler),
	}
}
