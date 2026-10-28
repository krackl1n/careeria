package preset

import (
	"net/http"

	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/middleware"
)

type MutatingConfig struct {
	ErrorHandler handler.ErrorHandler

	Authorizer middleware.Authorizer
}

func Mutating(config MutatingConfig) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middleware.Authorization(config.Authorizer, config.ErrorHandler),
	}
}
