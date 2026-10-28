package middleware

import "net/http"

type HeaderNormalizationConfig struct {
	RemoveRequestHeaders []string
	SetRequestHeaders    map[string]string
}

func HeaderNormalization(config HeaderNormalizationConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("header normalization middleware: nil handler")
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, header := range config.RemoveRequestHeaders {
				r.Header.Del(header)
			}

			for header, value := range config.SetRequestHeaders {
				r.Header.Set(header, value)
			}

			next.ServeHTTP(w, r)
		})
	}
}
