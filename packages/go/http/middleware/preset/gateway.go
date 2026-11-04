package preset

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/httperror"
	"github.com/krackl1n/careeria/http/middleware"
)

type GatewayConfig struct {
	Base BaseConfig

	TrustedProxies []netip.Prefix
	CORS           middleware.CORSConfig

	RateLimitRequests int
	RateLimitWindow   time.Duration

	BodyLimit        int64
	CompressionLevel int

	SecurityHeaders middleware.SecurityHeadersConfig
}

type GatewayDeps struct {
	Logger          *slog.Logger
	ErrorHandler    handler.ErrorHandler
	Registerer      prometheus.Registerer
	AllowOriginFunc func(*http.Request, string) bool
}

func (c GatewayConfig) Validate() error {
	if err := c.Base.Validate(); err != nil {
		return fmt.Errorf("validate base config: %w", err)
	}
	if c.RateLimitRequests < 0 {
		return fmt.Errorf("rate limit requests must not be negative")
	}
	if c.RateLimitRequests > 0 && c.RateLimitWindow <= 0 {
		return fmt.Errorf("rate limit window is required")
	}
	if c.BodyLimit <= 0 {
		return fmt.Errorf("body limit must be positive")
	}
	if c.CompressionLevel < -2 || c.CompressionLevel > 9 {
		return fmt.Errorf("compression level must be between -2 and 9")
	}
	return nil
}

func Gateway(
	config GatewayConfig,
	deps GatewayDeps,
) []func(http.Handler) http.Handler {
	result := Base(config.Base, BaseDeps{Logger: deps.Logger, ErrorHandler: deps.ErrorHandler, Registerer: deps.Registerer})
	result = append(result,
		middleware.RealIP(middleware.RealIPConfig{TrustedProxies: config.TrustedProxies}),
		middleware.HeaderNormalization(middleware.HeaderNormalizationConfig{
			RemoveRequestHeaders: []string{"X-Principal-Sub", "X-User-ID", "X-User-Roles", "X-Internal-Token", middleware.ClientIPHeader},
		}),
	)

	result = append(result,
		middleware.SecurityHeaders(config.SecurityHeaders),
		middleware.CORS(config.CORS, deps.AllowOriginFunc),
	)

	if config.RateLimitRequests > 0 && config.RateLimitWindow > 0 {
		result = append(result, httprate.LimitBy(config.RateLimitRequests, config.RateLimitWindow,
			func(r *http.Request) (string, error) {
				if clientIP := middleware.ClientIPFromRequest(r); clientIP != "" {
					return clientIP, nil
				}
				return "unknown", nil
			},
			httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
				h := deps.ErrorHandler
				if h == nil {
					h = handler.NewErrorHandler(nil)
				}
				h.Handle(w, r, httperror.New(http.StatusTooManyRequests, "rate_limited", http.StatusText(http.StatusTooManyRequests)))
			}),
		))
	}

	result = append(result, middleware.BodyLimit(config.BodyLimit, deps.ErrorHandler), chimiddleware.Compress(config.CompressionLevel))
	return result
}
