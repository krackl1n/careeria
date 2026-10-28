package middleware

import (
	"net/http"

	"github.com/krackl1n/careeria/ctxmeta"
	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/httperror"
)

type Authenticator interface {
	Authenticate(r *http.Request) (ctxmeta.Principal, error)
}

type AuthenticatorFunc func(*http.Request) (ctxmeta.Principal, error)

func (f AuthenticatorFunc) Authenticate(r *http.Request) (ctxmeta.Principal, error) {
	return f(r)
}

func Auth(authenticator Authenticator, errorHandler handler.ErrorHandler) func(http.Handler) http.Handler {
	if authenticator == nil {
		panic("auth middleware: nil authenticator")
	}

	if errorHandler == nil {
		errorHandler = handler.NewErrorHandler(nil)
	}

	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("auth middleware: nil handler")
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, err := authenticator.Authenticate(r)
			if err != nil || principal.SubjectID == "" {
				errorHandler.Handle(w, r, httperror.New(
					http.StatusUnauthorized,
					"unauthorized",
					http.StatusText(http.StatusUnauthorized),
				))
				return
			}
			ctx := ctxmeta.WithPrincipal(r.Context(), principal)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
