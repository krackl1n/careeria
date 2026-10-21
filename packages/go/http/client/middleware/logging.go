package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logging(logger *slog.Logger) func(http.RoundTripper) http.RoundTripper {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.RoundTripper) http.RoundTripper {
		if next == nil {
			next = http.DefaultTransport
		}

		return roundTripFunc(func(r *http.Request) (*http.Response, error) {
			startedAt := time.Now()

			resp, err := next.RoundTrip(r)

			duration := time.Since(startedAt)

			status := 0
			if resp != nil {
				status = resp.StatusCode
			}

			args := []any{
				"method", r.Method,
				"url", sanitizedURL(r),
				"host", host(r),
				"status", status,
				"duration", duration.String(),
				"duration_ms", float64(duration.Microseconds()) / 1000,
				"request_id", r.Header.Get(RequestIDHeader),
			}

			if err != nil {
				args = append(args, slog.Any("error", err))
				logger.ErrorContext(r.Context(), "http client request failed", args...)
				return resp, err
			}

			if status >= http.StatusInternalServerError {
				logger.ErrorContext(r.Context(), "http client request completed", args...)
				return resp, nil
			}

			logger.InfoContext(r.Context(), "http client request completed", args...)

			return resp, nil
		})
	}
}

func sanitizedURL(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}

	u := *r.URL
	u.RawQuery = ""

	return u.String()
}

func host(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}

	return r.URL.Host
}
