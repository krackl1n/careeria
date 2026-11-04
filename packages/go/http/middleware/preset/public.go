package preset

import (
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	chicors "github.com/go-chi/cors"

	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/middleware"
)

type PublicConfig struct {
	ErrorHandler handler.ErrorHandler

	CORS chicors.Options

	BodyLimit        int64
	CompressionLevel int
}

func Public(config PublicConfig) []func(http.Handler) http.Handler {
	compressionLevel := config.CompressionLevel
	if compressionLevel == 0 {
		compressionLevel = 5
	}

	return []func(http.Handler) http.Handler{
		chicors.Handler(config.CORS),
		middleware.BodyLimit(config.BodyLimit, config.ErrorHandler),
		chimiddleware.Compress(compressionLevel),
	}
}
