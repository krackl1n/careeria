package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/krackl1n/careeria/ctxmeta"
)

const (
	RequestIDHeader     = "X-Request-ID"
	CorrelationIDHeader = "X-Correlation-ID"
)

func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("request id middleware: nil handler")
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(RequestIDHeader)
			if !validRequestID(requestID) {
				requestID = generateRequestID()
			}

			r.Header.Set(RequestIDHeader, requestID)
			w.Header().Set(RequestIDHeader, requestID)

			correlationID := r.Header.Get(CorrelationIDHeader)
			if correlationID == "" {
				correlationID = requestID
				r.Header.Set(CorrelationIDHeader, correlationID)
			}

			ctx := ctxmeta.WithRequestID(r.Context(), requestID)
			ctx = ctxmeta.WithCorrelationID(ctx, correlationID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestIDFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}

	if requestID, ok := ctxmeta.RequestID(r.Context()); ok {
		return requestID
	}

	return r.Header.Get(RequestIDHeader)
}
func generateRequestID() string {
	var data [16]byte

	if _, err := rand.Read(data[:]); err != nil {
		return ""
	}

	return hex.EncodeToString(data[:])
}

func validRequestID(value string) bool {
	if len(value) < 8 || len(value) > 128 {
		return false
	}

	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z':
		case char >= 'A' && char <= 'Z':
		case char >= '0' && char <= '9':
		case char == '-':
		case char == '_':
		case char == '.':
		case char == ':':
		default:
			return false
		}
	}

	return true
}
