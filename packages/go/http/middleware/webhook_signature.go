package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/httperror"
)

type WebhookVerifier interface {
	Verify(r *http.Request, body []byte) error
}

type WebhookSignatureConfig struct {
	Verifier WebhookVerifier

	MaxBodyBytes int64
}

func WebhookSignature(config WebhookSignatureConfig, errorHandler handler.ErrorHandler) func(http.Handler) http.Handler {
	if config.Verifier == nil {
		panic("webhook signature middleware: nil verifier")
	}

	if config.MaxBodyBytes <= 0 {
		config.MaxBodyBytes = DefaultBodyLimit
	}

	if errorHandler == nil {
		errorHandler = handler.NewErrorHandler(nil)
	}

	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("webhook signature middleware: nil handler")
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := readBodyLimited(r, config.MaxBodyBytes)
			if err != nil {
				errorHandler.Handle(w, r, err)
				return
			}

			if err := config.Verifier.Verify(r, body); err != nil {
				errorHandler.Handle(w, r, httperror.Wrap(
					http.StatusUnauthorized,
					"invalid_webhook_signature",
					http.StatusText(http.StatusUnauthorized),
					err,
				))
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
		})
	}
}

func readBodyLimited(r *http.Request, limit int64) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if err != nil {
		return nil, err
	}

	if int64(len(body)) > limit {
		return nil, &http.MaxBytesError{Limit: limit}
	}

	if err := r.Body.Close(); err != nil {
		return nil, err
	}

	return body, nil
}
