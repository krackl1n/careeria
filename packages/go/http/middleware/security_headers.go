package middleware

import (
	"net/http"
	"strconv"
)

type SecurityHeadersConfig struct {
	ContentSecurityPolicy string
	ReferrerPolicy        string
	FrameOptions          string

	HSTSMaxAge            int
	HSTSIncludeSubdomains bool
	HSTSPreload           bool
}

func SecurityHeaders(config ...SecurityHeadersConfig) func(http.Handler) http.Handler {
	cfg := SecurityHeadersConfig{}
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.ReferrerPolicy == "" {
		cfg.ReferrerPolicy = "no-referrer"
	}

	if cfg.FrameOptions == "" {
		cfg.FrameOptions = "DENY"
	}

	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("security headers middleware: nil handler")
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := w.Header()

			header.Set("X-Content-Type-Options", "nosniff")
			header.Set("X-Frame-Options", cfg.FrameOptions)
			header.Set("Referrer-Policy", cfg.ReferrerPolicy)

			if cfg.ContentSecurityPolicy != "" {
				header.Set("Content-Security-Policy", cfg.ContentSecurityPolicy)
			}

			if r.TLS != nil && cfg.HSTSMaxAge > 0 {
				value := "max-age=" + strconv.Itoa(cfg.HSTSMaxAge)

				if cfg.HSTSIncludeSubdomains {
					value += "; includeSubDomains"
				}

				if cfg.HSTSPreload {
					value += "; preload"
				}

				header.Set("Strict-Transport-Security", value)
			}

			next.ServeHTTP(w, r)
		})
	}
}
