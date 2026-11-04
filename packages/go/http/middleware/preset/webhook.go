package preset

import (
	"net/http"

	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/middleware"
)

type WebhookConfig struct {
	ErrorHandler handler.ErrorHandler

	BodyLimit int64

	Signature middleware.WebhookSignatureConfig
}

func Webhook(config WebhookConfig) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middleware.BodyLimit(config.BodyLimit, config.ErrorHandler),
		middleware.WebhookSignature(config.Signature, config.ErrorHandler),
	}
}
