package preset

import (
	"log/slog"
	"net/http"

	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/middleware"
)

func Health(logger *slog.Logger, errorHandler handler.ErrorHandler) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middleware.Recover(logger, errorHandler),
		middleware.RequestID(),
	}
}
