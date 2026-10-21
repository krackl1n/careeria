package preset

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	clientmw "github.com/krackl1n/careeria/http/client/middleware"
)

type BaseConfig struct {
	Logger *slog.Logger

	ServiceName string

	MetricsRegisterer prometheus.Registerer
	MetricsNamespace  string
	MetricsSubsystem  string
	MetricsTarget     string

	TraceOptions []otelhttp.Option
}

func Base(cfg BaseConfig) []func(http.RoundTripper) http.RoundTripper {
	if cfg.ServiceName == "" {
		panic("http client preset base: empty service name")
	}

	return []func(http.RoundTripper) http.RoundTripper{
		clientmw.Tracing(cfg.TraceOptions...),
		clientmw.RequestID(clientmw.RequestIDConfig{}),
		clientmw.UserAgent(cfg.ServiceName),
		clientmw.Metrics(clientmw.MetricsConfig{
			Registerer: cfg.MetricsRegisterer,
			Namespace:  cfg.MetricsNamespace,
			Subsystem:  cfg.MetricsSubsystem,
			Target:     cfg.MetricsTarget,
		}),
		clientmw.Logging(cfg.Logger),
	}
}
