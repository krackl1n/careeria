package preset

import (
	"log/slog"
	"net/http"

	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/middleware"
)

type DebugConfig struct {
	Logger       *slog.Logger
	ErrorHandler handler.ErrorHandler

	Authenticator middleware.InternalAuthenticator
}

func Debug(config DebugConfig) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middleware.Recover(config.Logger, config.ErrorHandler),
		middleware.RequestID(),
		middleware.InternalAuth(config.Authenticator, config.ErrorHandler),
	}
}
