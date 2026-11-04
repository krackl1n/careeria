package metrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	cfg      Config
	registry *prometheus.Registry
}

func NewMetrics(cfg Config) (*Metrics, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate metrics config: %w", err)
	}

	registry := prometheus.NewRegistry()

	m := &Metrics{
		cfg:      cfg,
		registry: registry,
	}

	if !cfg.Enabled {
		return m, nil
	}

	if err := m.registerDefaultCollectors(); err != nil {
		return nil, fmt.Errorf("register default collectors: %w", err)
	}

	return m, nil
}

func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

func (m *Metrics) Namespace() string {
	return m.cfg.Namespace
}

func (m *Metrics) Enabled() bool {
	return m.cfg.Enabled
}

func (m *Metrics) Register(collector prometheus.Collector) error {
	if m == nil {
		return nil
	}

	if !m.cfg.Enabled {
		return nil
	}

	if err := m.registry.Register(collector); err != nil {
		return fmt.Errorf("register prometheus collector: %w", err)
	}

	return nil
}

func (m *Metrics) MustRegister(collectors ...prometheus.Collector) {
	if m == nil {
		return
	}

	if !m.cfg.Enabled {
		return
	}

	m.registry.MustRegister(collectors...)
}

func (m *Metrics) Unregister(collector prometheus.Collector) bool {
	if m == nil || !m.cfg.Enabled {
		return false
	}

	return m.registry.Unregister(collector)
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(
		m.registry,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	)
}

func (m *Metrics) registerDefaultCollectors() error {
	if err := m.registry.Register(collectors.NewGoCollector()); err != nil {
		return fmt.Errorf("register go collector: %w", err)
	}

	if err := m.registry.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{
		Namespace: m.cfg.Namespace,
	})); err != nil {
		return fmt.Errorf("register process collector: %w", err)
	}

	return nil
}
