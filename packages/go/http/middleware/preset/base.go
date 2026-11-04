package preset

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/krackl1n/careeria/http/handler"
	"github.com/krackl1n/careeria/http/middleware"
)

type BaseConfig struct {
	Metrics        middleware.MetricsConfig
	TraceOperation string
}

type BaseDeps struct {
	Logger       *slog.Logger
	ErrorHandler handler.ErrorHandler
	Registerer   prometheus.Registerer
	TraceOptions []otelhttp.Option
}

func (c BaseConfig) Validate() error {
	if c.TraceOperation == "" {
		return fmt.Errorf("base trace operation is required")
	}

	return nil
}

func Base(
	config BaseConfig,
	deps BaseDeps,
) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middleware.Recover(deps.Logger, deps.ErrorHandler),
		middleware.RequestID(),
		middleware.StructuredLogger(deps.Logger),
		middleware.Metrics(config.Metrics, deps.Registerer),
		otelhttp.NewMiddleware(config.TraceOperation, deps.TraceOptions...),
	}
}
