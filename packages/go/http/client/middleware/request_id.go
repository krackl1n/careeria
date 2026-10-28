package middleware

import (
	"net/http"

	"github.com/krackl1n/careeria/ctxmeta"
)

const (
	RequestIDHeader     = "X-Request-ID"
	CorrelationIDHeader = "X-Correlation-ID"
)

type RequestIDConfig struct {
	RequestIDHeader     string
	CorrelationIDHeader string
}

func RequestID(cfg RequestIDConfig) func(http.RoundTripper) http.RoundTripper {
	if cfg.RequestIDHeader == "" {
		cfg.RequestIDHeader = RequestIDHeader
	}
	if cfg.CorrelationIDHeader == "" {
		cfg.CorrelationIDHeader = CorrelationIDHeader
	}

	return func(next http.RoundTripper) http.RoundTripper {
		if next == nil {
			next = http.DefaultTransport
		}

		return roundTripFunc(func(r *http.Request) (*http.Response, error) {
			requestID, hasRequestID := ctxmeta.RequestID(r.Context())
			correlationID, hasCorrelationID := ctxmeta.CorrelationID(r.Context())
			setRequestID := r.Header.Get(cfg.RequestIDHeader) == "" && hasRequestID
			setCorrelationID := r.Header.Get(cfg.CorrelationIDHeader) == "" && hasCorrelationID
			if !setRequestID && !setCorrelationID {
				return next.RoundTrip(r)
			}

			clone := cloneRequest(r)
			if setRequestID {
				clone.Header.Set(cfg.RequestIDHeader, requestID)
			}
			if setCorrelationID {
				clone.Header.Set(cfg.CorrelationIDHeader, correlationID)
			}

			return next.RoundTrip(clone)
		})
	}
}
