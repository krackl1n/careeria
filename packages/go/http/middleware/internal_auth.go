package middleware

import (
	"net/http"

	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/httperror"
)

type InternalAuthenticator interface {
	AuthenticateInternal(r *http.Request) error
}

func InternalAuth(authenticator InternalAuthenticator, errorHandler handler.ErrorHandler) func(http.Handler) http.Handler {
	if authenticator == nil {
		panic("internal auth middleware: nil authenticator")
	}

	if errorHandler == nil {
		errorHandler = handler.NewErrorHandler(nil)
	}

	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("internal auth middleware: nil handler")
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := authenticator.AuthenticateInternal(r); err != nil {
				errorHandler.Handle(w, r, httperror.Wrap(
					http.StatusUnauthorized,
					"internal_unauthorized",
					http.StatusText(http.StatusUnauthorized),
					err,
				))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
