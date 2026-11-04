package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/felixge/httpsnoop"

	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/httperror"
)

func Recover(logger *slog.Logger, errorHandler handler.ErrorHandler) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	if errorHandler == nil {
		errorHandler = handler.NewErrorHandler(logger)
	}

	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("recover middleware: nil handler")
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrote := false

			wrapped := httpsnoop.Wrap(w, httpsnoop.Hooks{
				WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
					return func(code int) {
						wrote = true
						next(code)
					}
				},
				Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
					return func(data []byte) (int, error) {
						wrote = true
						return next(data)
					}
				},
			})

			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}

				logger.ErrorContext(
					r.Context(),
					"http panic recovered",
					"panic", recovered,
					"method", r.Method,
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr,
					"client_ip", ClientIPFromRequest(r),
					"request_id", RequestIDFromRequest(r),
					"stack", string(debug.Stack()),
				)

				if wrote {
					return
				}

				errorHandler.Handle(
					w,
					r,
					httperror.Wrap(
						http.StatusInternalServerError,
						"internal_server_error",
						http.StatusText(http.StatusInternalServerError),
						fmt.Errorf("panic recovered: %v", recovered),
					),
				)
			}()

			next.ServeHTTP(wrapped, r)
		})
	}
}
