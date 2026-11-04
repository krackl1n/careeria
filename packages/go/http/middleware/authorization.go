package middleware

import (
	"net/http"

	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/httperror"
)

type Authorizer interface {
	Authorize(r *http.Request) error
}

func Authorization(authorizer Authorizer, errorHandler handler.ErrorHandler) func(http.Handler) http.Handler {
	if authorizer == nil {
		panic("authorization middleware: nil authorizer")
	}

	if errorHandler == nil {
		errorHandler = handler.NewErrorHandler(nil)
	}

	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("authorization middleware: nil handler")
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := authorizer.Authorize(r); err != nil {
				errorHandler.Handle(w, r, httperror.Wrap(
					http.StatusForbidden,
					"forbidden",
					http.StatusText(http.StatusForbidden),
					err,
				))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
