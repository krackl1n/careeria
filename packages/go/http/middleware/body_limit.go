package middleware

import (
	"net/http"

	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/httperror"
)

const DefaultBodyLimit int64 = 1 << 20 // 1 MiB

func BodyLimit(limit int64, errorHandler handler.ErrorHandler) func(http.Handler) http.Handler {
	if limit <= 0 {
		limit = DefaultBodyLimit
	}

	if errorHandler == nil {
		errorHandler = handler.NewErrorHandler(nil)
	}

	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("body limit middleware: nil handler")
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > limit {
				errorHandler.Handle(w, r, httperror.New(
					http.StatusRequestEntityTooLarge,
					"payload_too_large",
					http.StatusText(http.StatusRequestEntityTooLarge),
				))
				return
			}

			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}

			next.ServeHTTP(w, r)
		})
	}
}
