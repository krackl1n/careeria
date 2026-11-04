package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/felixge/httpsnoop"
)

func StructuredLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("structured logger middleware: nil handler")
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()

			var metrics httpsnoop.Metrics
			panicked := true

			defer func() {
				duration := time.Since(startedAt)

				status := metrics.Code
				if status == 0 {
					status = http.StatusOK
				}

				if panicked {
					status = http.StatusInternalServerError
				}

				args := []any{
					"method", r.Method,
					"path", r.URL.Path,
					"query", r.URL.RawQuery,
					"status", status,
					"bytes", metrics.Written,
					"duration", duration.String(),
					"duration_ms", float64(duration.Microseconds()) / 1000,
					"remote_addr", r.RemoteAddr,
					"client_ip", ClientIPFromRequest(r),
					"user_agent", r.UserAgent(),
					"request_id", RequestIDFromRequest(r),
				}

				if panicked || status >= http.StatusInternalServerError {
					logger.ErrorContext(r.Context(), "http request completed", args...)
					return
				}

				logger.InfoContext(r.Context(), "http request completed", args...)
			}()

			metrics = httpsnoop.CaptureMetrics(next, w, r)
			panicked = false
		})
	}
}
