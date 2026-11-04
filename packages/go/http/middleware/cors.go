package middleware

import (
	"net/http"

	chicors "github.com/go-chi/cors"
)

type CORSConfig struct {
	AllowedOrigins     []string
	AllowedMethods     []string
	AllowedHeaders     []string
	ExposedHeaders     []string
	AllowCredentials   bool
	MaxAge             int
	OptionsPassthrough bool
	Debug              bool
}

func CORS(config CORSConfig, allowOriginFunc func(*http.Request, string) bool) func(http.Handler) http.Handler {
	return chicors.Handler(chicors.Options{
		AllowedOrigins:     config.AllowedOrigins,
		AllowedMethods:     config.AllowedMethods,
		AllowedHeaders:     config.AllowedHeaders,
		ExposedHeaders:     config.ExposedHeaders,
		AllowCredentials:   config.AllowCredentials,
		MaxAge:             config.MaxAge,
		OptionsPassthrough: config.OptionsPassthrough,
		Debug:              config.Debug,
		AllowOriginFunc:    allowOriginFunc,
	})
}
